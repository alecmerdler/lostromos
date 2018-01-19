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

package crwatcher_test

import (
	"testing"
	
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cr "github.com/wpengine/lostromos/crwatcher"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
)

func TestSetPhase(t *testing.T) {
	newStatus := cr.SetPhase(testStatus(), cr.PhaseApplying, "ResourcesCreating", "working on it")

	assert.Equal(t, string(cr.PhaseApplying), string(newStatus.Phase))
	assert.Equal(t, "ResourcesCreating", string(newStatus.Reason))
	assert.Equal(t, "working on it", newStatus.Message)
	assert.NotEqual(t, metav1.Now(), newStatus.LastUpdateTime)
	assert.NotEqual(t, metav1.Now(), newStatus.LastTransitionTime)
}

func TestStatusForEmptyStatus(t *testing.T) {
	status := cr.StatusFor(testResource())

	assert.Equal(t, cr.CustomResourceStatus{}, status)
}

func TestStatusForFilledStatus(t *testing.T) {
	expectedResource := testResource()
	expectedResource.Object["status"] = testStatusRaw()
	status := cr.StatusFor(expectedResource)

	assert.EqualValues(t, testStatus().Phase, status.Phase)
	assert.EqualValues(t, testStatus().Reason, status.Reason)
	assert.EqualValues(t, testStatus().Message, status.Message)
}

func testResource() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "dory",
			},
			"spec": map[string]interface{}{
				"Name": "Dory",
				"From": "Finding Nemo",
				"By":   "Disney",
			},
		},
	}
}

func testStatus() cr.CustomResourceStatus {
	return cr.CustomResourceStatus{
		Phase: cr.PhaseApplied,
		Reason: cr.ReasonApplySuccessful,
		Message: "some message",
		LastUpdateTime: metav1.Now(),
		LastTransitionTime: metav1.Now(),
	}
}

func testStatusRaw() map[string]interface{} {
	return map[string]interface{}{
		"phase": cr.PhaseApplied,
		"reason": cr.ReasonApplySuccessful,
		"message": "some message",
		"lastUpdateTime": metav1.Now().UTC(),
		"lastTransitionTime": metav1.Now().UTC(),
	}
}
