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

package controllers

import (
	"context"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/go-logr/logr"
	slacknotifyv1 "github.com/hamadakafu/slack-notify/api/v1"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SlackNotifyReconciler reconciles a SlackNotify object
type SlackNotifyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// ジョブを作ったりするので権限を指定している
// +kubebuilder:rbac:groups=slacknotify.kafuhamada.kubebuilder.io,resources=slacknotifies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=slacknotify.kafuhamada.kubebuilder.io,resources=slacknotifies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

func ignoreAlreadyExists(err error) error {
	if apierrs.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (r *SlackNotifyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("slacknotify", req.NamespacedName)

	// 1. Load the SlackNotify by name
	var slackNotify slacknotifyv1.SlackNotify
	if err := r.Get(ctx, req.NamespacedName, &slackNotify); err != nil {
		log.Error(err, "unable to fetch SlackNotify")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, ignoreNotFound(err)
	}

	// 2. finalizer の定義
	slackNotifyFinalizer := "finalizer.slacknotify.kafuhamada.kubebuilder.io"
	if slackNotify.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(slackNotify.ObjectMeta.Finalizers, slackNotifyFinalizer) {
			slackNotify.ObjectMeta.Finalizers = append(slackNotify.ObjectMeta.Finalizers, slackNotifyFinalizer)
			if err := r.Update(context.Background(), &slackNotify); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(slackNotify.ObjectMeta.Finalizers, slackNotifyFinalizer) {
			if err := r.deleteExternalResources(req, &slackNotify); err != nil {
				return ctrl.Result{}, err
			}
		}
		slackNotify.ObjectMeta.Finalizers = removeString(slackNotify.ObjectMeta.Finalizers, slackNotifyFinalizer)
		if err := r.Update(context.Background(), &slackNotify); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. create SlackNotify
	constructJob := func(sn *slacknotifyv1.SlackNotify) (*batch.Job, error) {
		// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
		job := &batch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        sn.Name,
				Namespace:   sn.Namespace,
			},
			Spec: batch.JobSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						RestartPolicy: v1.RestartPolicyOnFailure,
						Containers: []v1.Container{
							v1.Container{
								Name:  sn.Name,
								Image: "technosophos/slack-notify",
								Env: []v1.EnvVar{
									v1.EnvVar{
										Name:  "SLACK_WEBHOOK",
										Value: sn.Spec.SlackWebhook,
									},
									v1.EnvVar{
										Name:  "SLACK_MESSAGE",
										Value: sn.Spec.Message,
									},
								},
								// TODO: SLACK_COLOR
							},
						},
					},
				},
			},
		}
		if err := ctrl.SetControllerReference(sn, job, r.Scheme); err != nil {
			return nil, err
		}
		return job, nil
	}
	// actually make the job...
	job, err := constructJob(&slackNotify)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		// don't bother requeuing until we get a change to the spec
		return ctrl.Result{}, nil
	}

	if err := r.Create(ctx, job); ignoreAlreadyExists(err) != nil {
		log.Error(err, "unable to create Job for SlackNotify", "job", job)
		return ctrl.Result{}, err
	}

	log.V(1).Info("created Job for SlackNofity run", "job", job)

	// we'll requeue once we see the running job, and update our status
	return ctrl.Result{}, nil
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = slacknotifyv1.GroupVersion.String()
)

func (r *SlackNotifyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&batch.Job{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		job := rawObj.(*batch.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		// ...make sure it's a CronJob...
		if owner.APIVersion != apiGVStr || owner.Kind != "SlackNotify" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&slacknotifyv1.SlackNotify{}).
		Owns(&batch.Job{}).
		Complete(r)
}

func (r *SlackNotifyReconciler) deleteExternalResources(req ctrl.Request, sn *slacknotifyv1.SlackNotify) error {
	ctx := context.Background()
	log.Info("delete called")
	var job batch.Job
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		log.Error(err, "unable to fetch Job")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ignoreNotFound(err)
	}
	if err := r.Delete(ctx, &job, client.PropagationPolicy(metav1.DeletePropagationBackground)); ignoreNotFound(err) != nil {
		log.Error(err, "unable to delete old failed job", "job", job)
	}
	return nil
}

func containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
func removeString(slice []string, str string) (result []string) {
	for _, item := range slice {
		if item == str {
			continue
		}
		result = append(result, item)
	}
	return
}
