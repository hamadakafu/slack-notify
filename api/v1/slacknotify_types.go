/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SlackNotifySpec defines the desired state of SlackNotify
type SlackNotifySpec struct {
	Message      string `json:"message"`
	SlackWebhook string `json:"slack_webhook"`
}

// SlackNotifyStatus defines the observed state of SlackNotify
type SlackNotifyStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name=MESSAGE,priority=1,type=string,JSONPath=`.spec.message`
// +kubebuilder:printcolumn:name=SLACK_WEBHOOK,priority=1,type=string,JSONPath=`.spec.slack_webhook`

// SlackNotify is the Schema for the slacknotifies API
type SlackNotify struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SlackNotifySpec   `json:"spec,omitempty"`
	Status            SlackNotifyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SlackNotifyList contains a list of SlackNotify
type SlackNotifyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SlackNotify `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SlackNotify{}, &SlackNotifyList{})
}
