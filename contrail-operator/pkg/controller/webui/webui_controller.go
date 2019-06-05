package webui

import (
	"context"

	contrailoperatorsv1alpha1 "github.com/operators/contrail-operator/pkg/apis/contrailoperators/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_webui")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new WebUI Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileWebUI{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("webui-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &contrailoperatorsv1alpha1.InfraVars{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner WebUI
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &contrailoperatorsv1alpha1.InfraVars{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileWebUI{}

// ReconcileWebUI reconciles a WebUI object
type ReconcileWebUI struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

var contrail_registry, contrail_tag string

func (r *ReconcileWebUI) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling WebUI")
	instance := &contrailoperatorsv1alpha1.InfraVars{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	contrail_registry = instance.Spec.ContrailRegistry
	contrail_tag = instance.Spec.ContrailTag
	ds := newDSForCR(instance)

	if err := controllerutil.SetControllerReference(instance, ds, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	found := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DS", "DS.Namespace", ds.Namespace, "DS.Name", ds.Name)
		err = r.client.Create(context.TODO(), ds)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}
	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: DS already exists", "DS.Namespace", found.Namespace, "DS.Name", found.Name)

	return reconcile.Result{}, nil
}

func newDSForCR(cr *contrailoperatorsv1alpha1.InfraVars) *appsv1.DaemonSet{
    labels := map[string]string{
								"app": "webui",
							}
		return &appsv1.DaemonSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "webui" + "-ds",
							Namespace: cr.Namespace,
							Labels:    labels,
						},
						Spec: appsv1.DaemonSetSpec{
							Selector: &metav1.LabelSelector{MatchLabels: labels},
							Template: corev1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "webui" + "-pod-template",
									Namespace: cr.Namespace,
									Labels:    labels,
								},
								Spec: corev1.PodSpec{
									HostNetwork: true,
									NodeSelector: map[string]string{
													"node-role.kubernetes.io/master": "",
									},
									Tolerations: []corev1.Toleration{
										{
											Key: "node.kubernetes.io/not-ready",
											Operator: "Exists",
										},
										{
											Key: "node.kubernetes.io/master",
											Operator: "Exists",
										},
									},
									InitContainers: initContainersForDS(cr),
									Containers: containersForDS(cr),
									Volumes: volumesForDS(),
								},
							},
						},
		}
}

func initContainersForDS(cr *contrailoperatorsv1alpha1.InfraVars) []corev1.Container{

	return []corev1.Container{
		{
			Name:    		"contrail-node-init",
			Image:   		contrail_registry+"/contrail-node-init"+contrail_tag,
			SecurityContext:	&corev1.SecurityContext{
							Privileged: func(b bool) *bool { return &b }(true),
			},
			Env:			[]corev1.EnvVar{
						{
							Name: "IPTABLES_CHAIN",
							Value: "INPUT",
						},
						{
							Name: "CONFIGURE_IPTABLES",
							Value: "true",
						},
						{
							Name: "NODE_TYPE",
							Value: "webui",
						},
						{
							Name: "CONTRAIL_STATUS_IMAGE",
							Value: "/contrail-status",
						},
			},
			EnvFrom:		[]corev1.EnvFromSource{
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: "contrail-conf-env"},
								},
							},
			},
			VolumeMounts:		[]corev1.VolumeMount{
						{
							MountPath: "/host/usr/bin",
							Name: "host-usr-bin",
						},
			},
	},
}
}

func containersForDS(cr *contrailoperatorsv1alpha1.InfraVars) []corev1.Container{
	return []corev1.Container{
	{
		Name:			"contrail-controller-webui-job",
		Image:   		contrail_registry+"/contrail-controller-webui-job"+contrail_tag,
		ImagePullPolicy: "IfNotPresent",
		SecurityContext:	&corev1.SecurityContext{
						Privileged: func(b bool) *bool { return &b }(true),
		},
		EnvFrom:		[]corev1.EnvFromSource{
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "contrail-conf-env"},
				},
			},
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "contrail-configzk-conf-env"},
				},
			},
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "contrail-kubernetes-conf-env"},
				},
			},
		},
		VolumeMounts:		[]corev1.VolumeMount{
				{
					MountPath: "/mnt",
					Name: "docker-unix-socket",
				},
				{
					MountPath: "/var/log/contrail",
					Name: "webui-logs",
				},
				{
					MountPath: "/tmp/serviceaccount",
					Name: "ssl-secret-gen-token",
				},
		},
	},
	{
		Name:			"contrail-controller-webui-web",
		Image:   		contrail_registry+"/contrail-controller-webui-web"+contrail_tag,
		ImagePullPolicy: "IfNotPresent",
		SecurityContext:	&corev1.SecurityContext{
						Privileged: func(b bool) *bool { return &b }(true),
		},
		EnvFrom:		[]corev1.EnvFromSource{
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "contrail-conf-env"},
				},
			},
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "contrail-configzk-conf-env"},
				},
			},
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "contrail-kubernetes-conf-env"},
				},
			},
		},
		VolumeMounts:		[]corev1.VolumeMount{
				{
					MountPath: "/var/log/contrail",
					Name: "webui-logs",
				},
				{
					MountPath: "/tmp/serviceaccount",
					Name: "ssl-secret-gen-token",
				},
				{
					MountPath: "/etc/localtime",
					Name: "localtime",
				},
		},
	},
}
}

func volumesForDS() []corev1.Volume{
	return []corev1.Volume{
		{
			Name: "webui-logs",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log/contrail/webui",
				},
			},
		},
		{
			Name: "docker-unix-socket",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/run",
				},
			},
		},
		{
			Name: "host-usr-bin",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/usr/bin",
				},
			},
		},
		{
			Name: "ssl-secret-gen-token",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource {
					SecretName: "contrail-kubernetes-token",
				},
			},
		},
		{
			Name: "host-var-lib",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib",
				},
			},
		},
		{
			Name: "localtime",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/localtime",
				},
			},
		},
	}
}
