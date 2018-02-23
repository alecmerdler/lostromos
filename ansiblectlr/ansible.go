// Copyright 2017 the lostromos Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ansiblectlr

import (
	"os/exec"
	"path"
	"strconv"

	crw "github.com/wpengine/lostromos/crwatcher"
	"github.com/wpengine/lostromos/metrics"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// Controller is a crwatcher.ResourceController that works with Ansible to deploy
// Ansible Playbook Bundles (APB) into k8s providing a CustomResource as value data to the resources
type Controller struct {
	apbDir         string
	logger         *zap.SugaredLogger
	resourceClient dynamic.ResourceInterface
}

// NewController will return a configured Ansible controller
func NewController(apbDir string, resourceClient dynamic.ResourceInterface, logger *zap.SugaredLogger) *Controller {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Controller{
		apbDir:         apbDir,
		resourceClient: resourceClient,
		logger:         logger,
	}
}

// ResourceAdded is called when a custom resource is created and will invoke the `provision` playbook
func (c Controller) ResourceAdded(r *unstructured.Unstructured) {
	metrics.TotalEvents.Inc()
	c.logger.Infow("resource added", "resource", r.GetName())

	var cmdOut []byte
	var err error

	// TODO(alecmerdler): Pass values from custom resource `spec` as `--extra-vars`
	// extraVars := map[string]interface{}{"namespace": r.GetNamespace()}
	// for k, v := range r.Object["spec"].(map[string]interface{}) {
	// 	extraVars[k] = v
	// }
	// jsonValues, _ := json.Marshal(extraVars)
	cmdArgs := append([]string{path.Join(c.apbDir, "provision.yml")}, extraVarsFrom(r)...)

	// FIXME(alecmerdler): Debugging
	cmdOut, _ = exec.Command("echo", append([]string{"ansible-playbook", path.Join(c.apbDir, "provision.yml")}, extraVarsFrom(r)...)...).Output()
	c.logger.Infof("exec'd command: %s", string(cmdOut))

	// FIXME(alecmerdler): Set `ownerReferences` on created resources
	// if cmdOut, err = exec.Command("ansible-playbook", path.Join(c.apbDir, "provision.yml"), "--extra-vars", "'"+string(jsonValues)+"'").Output(); err != nil {
	if cmdOut, err = exec.Command("ansible-playbook", cmdArgs...).Output(); err != nil {
		c.logger.Errorw("failed to create resource", "error", err, "message", string(cmdOut), "resource", r.GetName())
		c.updateCRStatus(r, crw.PhaseFailed, crw.ReasonApplyFailed, string(cmdOut))
		return
	}
	metrics.CreatedReleases.Inc()
	metrics.ManagedReleases.Inc()

	c.updateCRStatus(r, crw.PhaseApplied, crw.ReasonApplySuccessful, string(cmdOut))
}

// ResourceUpdated is called when a custom resource is updated and will invoke the `update` playbook
// NOTE: `Update` method is not documented, found here (https://github.com/openshift/ansible-service-broker/blob/master/pkg/broker/broker.go#L1059)
func (c Controller) ResourceUpdated(oldR, newR *unstructured.Unstructured) {
	metrics.TotalEvents.Inc()
	c.logger.Infow("resource updated", "resource", newR.GetName())

	var cmdOut []byte
	var err error

	cmdArgs := append([]string{path.Join(c.apbDir, "update.yml")}, extraVarsFrom(newR)...)

	if cmdOut, err = exec.Command("ansible-playbook", cmdArgs...).Output(); err != nil {
		c.logger.Errorw("failed to update resource", "error", err, "message", string(cmdOut), "resource", newR.GetName())
		c.updateCRStatus(newR, crw.PhaseFailed, crw.ReasonApplyFailed, string(cmdOut))
		return
	}
	metrics.UpdatedReleases.Inc()

	c.updateCRStatus(newR, crw.PhaseApplied, crw.ReasonApplySuccessful, string(cmdOut))
}

// ResourceDeleted is called when a custom resource is deleted and will invoke the `deprovision` playbook
func (c Controller) ResourceDeleted(r *unstructured.Unstructured) {
	metrics.TotalEvents.Inc()
	c.logger.Infow("resource deleted", "resource", r.GetName())

	var cmdOut []byte
	var err error

	cmdArgs := append([]string{path.Join(c.apbDir, "deprovision.yml")}, extraVarsFrom(r)...)

	if cmdOut, err = exec.Command("ansible-playbook", cmdArgs...).Output(); err != nil {
		metrics.DeleteFailures.Inc()
		c.logger.Errorw("failed to delete resource", "error", err, "message", string(cmdOut), "resource", r.GetName())
		return
	}
	metrics.DeletedReleases.Inc()
	metrics.ManagedReleases.Dec()
}

func (c Controller) updateCRStatus(r *unstructured.Unstructured, phase crw.ResourcePhase, reason crw.ConditionReason, message string) (*unstructured.Unstructured, error) {
	updatedResource := r.DeepCopy()
	status := crw.StatusFor(r)
	status.SetPhase(phase, reason, message)
	status.SetPodStatuses(map[string][]string{
		"ready":   []string{"my-test-example-1", "my-test-example-2"},
		"waiting": []string{"my-test-example-3"},
	})
	statusMap, err := status.ToMap()
	if err != nil {
		return nil, err
	}
	updatedResource.Object["status"] = statusMap
	return c.resourceClient.Update(updatedResource)
}

// FIXME(alecmerdler): Only grabs `size` and `namespace` fields
func extraVarsFrom(r *unstructured.Unstructured) []string {
	return []string{
		"-e", "namespace=" + r.GetNamespace(),
		"-e", "size=" + strconv.FormatInt(r.Object["spec"].(map[string]interface{})["size"].(int64), 10),
	}
}
