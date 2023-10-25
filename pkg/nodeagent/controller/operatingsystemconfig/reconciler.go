// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operatingsystemconfig

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	extensionsv1alpha1helper "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1/helper"
	"github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/executor"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/nodeagent/apis/config"
	nodeagentv1alpha1 "github.com/gardener/gardener/pkg/nodeagent/apis/config/v1alpha1"
	"github.com/gardener/gardener/pkg/nodeagent/dbus"
	"github.com/gardener/gardener/pkg/utils/flow"
)

const (
	dataKeyOperatingSystemConfig             = "osc.yaml"
	lastAppliedOperatingSystemConfigFilePath = nodeagentv1alpha1.BaseDir + "/last-applied-osc.yaml"
)

// Reconciler decodes the OperatingSystemConfig resources from secrets and applies the systemd units and files to the
// node.
type Reconciler struct {
	Client   client.Client
	Config   config.OperatingSystemConfigControllerConfig
	Recorder record.EventRecorder
	DBus     dbus.DBus
	FS       afero.Afero
	NodeName string
}

// Reconcile decodes the OperatingSystemConfig resources from secrets and applies the systemd units and files to the
// node.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	ctx, cancel := controllerutils.GetMainReconciliationContext(ctx, controllerutils.DefaultReconciliationTimeout)
	defer cancel()

	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, request.NamespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("Object is gone, stop reconciling")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error retrieving object from store: %w", err)
	}

	node := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: r.NodeName}}
	node.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Node"))
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(node), node); err != nil {
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, fmt.Errorf("unable to fetch node %q: %w", r.NodeName, err)
		}
		node = nil
	}

	osc, oscRaw, oscChecksum, err := extractOSCFromSecret(secret)
	if err != nil {
		return reconcile.Result{}, err
	}

	oscChanges, err := computeOperatingSystemConfigChanges(r.FS, osc)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to calculate the OSC changes: %w", err)
	}

	if node != nil && node.Annotations[executor.AnnotationKeyChecksum] == oscChecksum {
		log.Info("Configuration on this node is up to date, nothing to be done")
		return reconcile.Result{}, nil
	}

	log.Info("Applying new or changed files")
	if err := r.applyChangedFiles(log, oscChanges.files.changed); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed applying changed files: %w", err)
	}

	log.Info("Applying new or changed units")
	if err := r.applyChangedUnits(ctx, log, oscChanges.units.changed); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed applying changed units: %w", err)
	}

	log.Info("Removing no longer needed units")
	if err := r.removeDeletedUnits(ctx, log, node, oscChanges.units.deleted); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed removing deleted units: %w", err)
	}

	log.Info("Reloading systemd daemon")
	if err := r.DBus.DaemonReload(ctx); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed reloading systemd daemon: %w", err)
	}

	log.Info("Executing unit commands (start/stop)")
	if err := r.executeUnitCommands(ctx, log, node, oscChanges.units.changed); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed executing unit commands: %w", err)
	}

	log.Info("Removing no longer needed files")
	if err := r.removeDeletedFiles(log, oscChanges.files.deleted); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed removing deleted files: %w", err)
	}

	log.Info("Successfully applied operating system config",
		"changedFiles", len(oscChanges.files.changed),
		"deletedFiles", len(oscChanges.files.deleted),
		"changedUnits", len(oscChanges.units.changed),
		"deletedUnits", len(oscChanges.units.deleted),
	)

	log.Info("Persisting current operating system config as 'last-applied' file to the disk", "path", lastAppliedOperatingSystemConfigFilePath)
	if err := r.FS.WriteFile(lastAppliedOperatingSystemConfigFilePath, oscRaw, 0644); err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to write current OSC to file path %q: %w", lastAppliedOperatingSystemConfigFilePath, err)
	}

	if node == nil {
		log.Info("Waiting for Node to get registered by kubelet, requeuing")
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	r.Recorder.Event(node, corev1.EventTypeNormal, "OSCApplied", "Operating system config has been applied successfully")

	patch := client.MergeFrom(node.DeepCopy())
	metav1.SetMetaDataAnnotation(&node.ObjectMeta, v1beta1constants.LabelWorkerKubernetesVersion, r.Config.KubernetesVersion.String())
	metav1.SetMetaDataAnnotation(&node.ObjectMeta, executor.AnnotationKeyChecksum, oscChecksum)
	return reconcile.Result{RequeueAfter: r.Config.SyncPeriod.Duration}, r.Client.Patch(ctx, node, patch)
}

var (
	etcSystemdSystem                   = path.Join("/", "etc", "systemd", "system")
	defaultFilePermissions os.FileMode = 0600
)

func (r *Reconciler) applyChangedFiles(log logr.Logger, files []extensionsv1alpha1.File) error {
	tmpDir, err := r.FS.TempDir("", "gardener-node-agent-*")
	if err != nil {
		return fmt.Errorf("unable to create temporary directory: %w", err)
	}
	defer func() { utilruntime.HandleError(r.FS.RemoveAll(tmpDir)) }()

	for _, file := range files {
		if file.Content.Inline == nil {
			continue
		}

		if err := r.FS.MkdirAll(filepath.Dir(file.Path), fs.ModeDir); err != nil {
			return fmt.Errorf("unable to create directory %q: %w", file.Path, err)
		}

		permissions := defaultFilePermissions
		if file.Permissions != nil {
			permissions = fs.FileMode(*file.Permissions)
		}

		data, err := extensionsv1alpha1helper.Decode(file.Content.Inline.Encoding, []byte(file.Content.Inline.Data))
		if err != nil {
			return fmt.Errorf("unable to decode data of file %q: %w", file.Path, err)
		}

		tmpFilePath := filepath.Join(tmpDir, filepath.Base(file.Path))
		if err := r.FS.WriteFile(tmpFilePath, data, permissions); err != nil {
			return fmt.Errorf("unable to create temporary file %q: %w", tmpFilePath, err)
		}

		if err := r.FS.Rename(tmpFilePath, file.Path); err != nil {
			return fmt.Errorf("unable to rename temporary file %q to %q: %w", tmpFilePath, file.Path, err)
		}

		log.Info("Successfully applied new or changed file", "path", file.Path)
	}

	return nil
}

func (r *Reconciler) removeDeletedFiles(log logr.Logger, files []extensionsv1alpha1.File) error {
	for _, file := range files {
		if err := r.FS.Remove(file.Path); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
			return fmt.Errorf("unable to delete no longer needed file %q: %w", file.Path, err)
		}

		log.Info("Successfully removed no longer needed file", "path", file.Path)
	}

	return nil
}

func (r *Reconciler) applyChangedUnits(ctx context.Context, log logr.Logger, units []changedUnit) error {
	for _, unit := range units {
		unitFilePath := path.Join(etcSystemdSystem, unit.Name)

		if unit.Content != nil {
			oldUnitContent, err := r.FS.ReadFile(unitFilePath)
			if err != nil && !errors.Is(err, afero.ErrFileNotFound) {
				return fmt.Errorf("unable to read existing unit file %q for %q: %w", unitFilePath, unit.Name, err)
			}

			newUnitContent := []byte(*unit.Content)
			if !bytes.Equal(newUnitContent, oldUnitContent) {
				if err := r.FS.WriteFile(unitFilePath, newUnitContent, defaultFilePermissions); err != nil {
					return fmt.Errorf("unable to write unit file %q for %q: %w", unitFilePath, unit.Name, err)
				}
				log.Info("Successfully applied new or changed unit file", "path", unitFilePath)
			}

			// ensure file permissions are restored in case somebody changed them manually
			if err := r.FS.Chmod(unitFilePath, defaultFilePermissions); err != nil {
				return fmt.Errorf("unable to ensure permissions for unit file %q for %q: %w", unitFilePath, unit.Name, err)
			}
		}

		dropInDirectory := unitFilePath + ".d"

		if len(unit.DropIns) == 0 {
			if err := r.FS.RemoveAll(dropInDirectory); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
				return fmt.Errorf("unable to delete systemd drop-in folder for unit %q: %w", unit.Name, err)
			}
		} else {
			if err := r.FS.MkdirAll(dropInDirectory, fs.ModeDir); err != nil {
				return fmt.Errorf("unable to create drop-in directory %q for unit %q: %w", dropInDirectory, unit.Name, err)
			}

			for _, dropIn := range unit.dropIns.changed {
				dropInFilePath := path.Join(dropInDirectory, dropIn.Name)

				oldDropInContent, err := r.FS.ReadFile(dropInFilePath)
				if err != nil && !errors.Is(err, afero.ErrFileNotFound) {
					return fmt.Errorf("unable to read existing drop-in file %q for unit %q: %w", dropInFilePath, unit.Name, err)
				}

				newDropInContent := []byte(dropIn.Content)
				if !bytes.Equal(newDropInContent, oldDropInContent) {
					if err := r.FS.WriteFile(dropInFilePath, newDropInContent, defaultFilePermissions); err != nil {
						return fmt.Errorf("unable to write drop-in file %q for unit %q: %w", dropInFilePath, unit.Name, err)
					}
					log.Info("Successfully applied new or changed drop-in file for unit", "path", dropInFilePath, "unit", unit.Name)
				}

				// ensure file permissions are restored in case somebody changed them manually
				if err := r.FS.Chmod(dropInFilePath, defaultFilePermissions); err != nil {
					return fmt.Errorf("unable to ensure permissions for drop-in file %q for unit %q: %w", unitFilePath, unit.Name, err)
				}
			}

			for _, dropIn := range unit.dropIns.deleted {
				dropInFilePath := path.Join(dropInDirectory, dropIn.Name)
				if err := r.FS.Remove(dropInFilePath); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
					return fmt.Errorf("unable to delete drop-in file %q for unit %q: %w", dropInFilePath, unit.Name, err)
				}
				log.Info("Successfully removed no longer needed drop-in file for unit", "path", dropInFilePath, "unitName", unit.Name)
			}
		}

		if pointer.BoolDeref(unit.Enable, true) {
			if err := r.DBus.Enable(ctx, unit.Name); err != nil {
				return fmt.Errorf("unable to enable unit %q: %w", unit.Name, err)
			}
			log.Info("Successfully enabled unit", "unitName", unit.Name)
		} else {
			if err := r.DBus.Disable(ctx, unit.Name); err != nil {
				return fmt.Errorf("unable to disable unit %q: %w", unit.Name, err)
			}
			log.Info("Successfully disabled unit", "unitName", unit.Name)
		}
	}

	return nil
}

func (r *Reconciler) removeDeletedUnits(ctx context.Context, log logr.Logger, node client.Object, units []extensionsv1alpha1.Unit) error {
	for _, unit := range units {
		if err := r.DBus.Disable(ctx, unit.Name); err != nil {
			return fmt.Errorf("unable to disable deleted unit %q: %w", unit.Name, err)
		}

		if err := r.DBus.Stop(ctx, r.Recorder, node, unit.Name); err != nil {
			return fmt.Errorf("unable to stop deleted unit %q: %w", unit.Name, err)
		}

		if err := r.FS.Remove(path.Join(etcSystemdSystem, unit.Name)); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
			return fmt.Errorf("unable to delete systemd unit file of deleted unit %q: %w", unit.Name, err)
		}

		if err := r.FS.RemoveAll(path.Join(etcSystemdSystem, unit.Name+".d")); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
			return fmt.Errorf("unable to delete systemd drop-in folder of deleted unit %q: %w", unit.Name, err)
		}

		log.Info("Successfully removed no longer needed unit", "unitName", unit.Name)
	}

	return nil
}

func (r *Reconciler) executeUnitCommands(ctx context.Context, log logr.Logger, node client.Object, units []changedUnit) error {
	var fns []flow.TaskFn

	for _, u := range units {
		unit := u

		fns = append(fns, func(ctx context.Context) error {
			if !pointer.BoolDeref(unit.Enable, true) || (unit.Command != nil && *unit.Command == extensionsv1alpha1.CommandStop) {
				if err := r.DBus.Stop(ctx, r.Recorder, node, unit.Name); err != nil {
					return fmt.Errorf("unable to stop unit %q: %w", unit.Name, err)
				}
				log.Info("Successfully stopped unit", "unitName", unit.Name)
			} else {
				if err := r.DBus.Restart(ctx, r.Recorder, node, unit.Name); err != nil {
					return fmt.Errorf("unable to restart unit %q: %w", unit.Name, err)
				}
				log.Info("Successfully restarted unit", "unitName", unit.Name)
			}

			return nil
		})
	}

	return flow.Parallel(fns...)(ctx)
}