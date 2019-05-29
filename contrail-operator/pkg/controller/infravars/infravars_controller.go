package infravars

import (
	"context"
	"fmt"
	"strings"

	contrailoperatorsv1alpha1 "github.com/operators/contrail-operator/pkg/apis/contrailoperators/v1alpha1"

	corev1 "k8s.io/api/core/v1"
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

var log = logf.Log.WithName("controller_infravars")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new InfraVars Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileInfraVars{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("infravars-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource InfraVars
	err = c.Watch(&source.Kind{Type: &contrailoperatorsv1alpha1.InfraVars{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner InfraVars
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &contrailoperatorsv1alpha1.InfraVars{},
	})

	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileInfraVars{}

// ReconcileInfraVars reconciles a InfraVars object
type ReconcileInfraVars struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}


func (r *ReconcileInfraVars) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling InfraVars")
	instance := &contrailoperatorsv1alpha1.InfraVars{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	nodeList := &corev1.NodeList{}
  if err = r.client.List(context.TODO(), nil, nodeList); err != nil {
        return reconcile.Result{}, err
  }

	var masterNodes []string

  for _, node := range nodeList.Items {
				for _, taint := range node.Spec.Taints {
					if string(taint.Key) == "node-role.kubernetes.io/master" {
						fmt.Println("Got master node :", node.Status.Addresses[0].Address)
						masterNodes = append(masterNodes, node.Status.Addresses[0].Address)
					}
				}
	}
	contrailNodes := strings.Join(masterNodes, ",")

	fmt.Println("Contrail Nodes :", contrailNodes)

  var apiServer string
	var apiServerPort string

	epList := &corev1.EndpointsList{}
	if err = r.client.List(context.TODO(), nil, epList); err != nil {
				return reconcile.Result{}, err
	}
	for _, ep := range epList.Items {
		if string(ep.Name) == "kubernetes" {
			fmt.Println("K8s service and port : ", ep.Subsets[0].Addresses[0].IP, ep.Subsets[0].Ports[0].Port)
			apiServer = ep.Subsets[0].Addresses[0].IP
			apiServerPort = string(ep.Subsets[0].Ports[0].Port)
		}
	}

	envCm := newEnvCm(instance, contrailNodes)
	kubeManagerCm := newKubeManagerCm(instance, apiServer, apiServerPort)
	kubernetesCm := newKubernetesCm(instance, apiServer, apiServerPort)
	configZkCm := newConfigZkCm(instance)
	analyticsZkCm := newAnalyticsZkCm(instance)
	analyticsDbCm := newAnalyticsDbCm(instance, contrailNodes)
	configDbCm := newConfigDbCm(instance, contrailNodes)
	rabbitCm := newRabbitCm(instance)
	nodemgrCm := newNodeMgrCmForDS(instance)
	secretK8s := newSecretK8s(instance)

	mainCm := []*corev1.ConfigMap{envCm, kubeManagerCm, kubernetesCm, configZkCm,analyticsZkCm, analyticsDbCm, configDbCm, rabbitCm, nodemgrCm}

	for _, cm := range mainCm {
	if err := controllerutil.SetControllerReference(instance, cm, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	found_cm := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, found_cm)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.client.Create(context.TODO(), cm)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
	reqLogger.Info("Skip reconcile: ConfigMap already exists", "ConfigMap.Namespace", found_cm.Namespace, "ConfigMap.Name", found_cm.Name)
	}
	}

	if err := controllerutil.SetControllerReference(instance, secretK8s, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	found_secret := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: secretK8s.Name, Namespace: secretK8s.Namespace}, found_secret)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Secret", "Secret.Namespace", secretK8s.Namespace, "Secret.Name", secretK8s.Name)
		err = r.client.Create(context.TODO(), secretK8s)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
	reqLogger.Info("Skip reconcile: Secret already exists", "Secret.Namespace", found_secret.Namespace, "Secret.Name", found_secret.Name)
	}

	return reconcile.Result{}, nil
}

func getContrailNodes(nodes []corev1.Node) []string{
	var nodeNames []string
	for _, node := range nodes {
		nodeLabels := node.ObjectMeta.Labels
		for _, label := range nodeLabels {
			if "opencontrail.org/controller" == label {
				nodeNames = append(nodeNames, node.ObjectMeta.Name)
			}
		}
	}
	return nodeNames
}

func newSecretK8s(cr *contrailoperatorsv1alpha1.InfraVars) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:           "contrail-kubernetes-token",
				Namespace:      cr.Namespace,
				Annotations:    map[string]string {
					"kubernetes.io/service-account.name": "contrail-operator",
				},
			},
			Type:	"kubernetes.io/service-account-token",
		}
}

func newNodeMgrCmForDS(cr *contrailoperatorsv1alpha1.InfraVars) *corev1.ConfigMap{
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:           "contrail-nodemgr-conf-env",
			Namespace:      cr.Namespace,
		},
		Data: map[string]string{
			"DOCKER_HOST": "unix://mnt/docker.sock",
		},
	}
}

func newEnvCm(cr *contrailoperatorsv1alpha1.InfraVars, contrailNodes string) *corev1.ConfigMap{
//	contrailNodes := cr.Spec.ContrailMasters
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:		"contrail-conf-env",
			Namespace:	cr.Namespace,
		},
		Data: map[string]string{
			"AAA_MODE":	"no-auth",
			"AUTH_MODE":	"noauth",
			"CLOUD_ORCHESTRATOR": "kubernetes",
			"CONFIG_NODES": contrailNodes,
			"CONFIGDB_NODES": contrailNodes,
			"CONTROL_NODES": contrailNodes,
			"CONTROLLER_NODES": contrailNodes,
			"KAFKA_NODES": contrailNodes,
			"LOG_LEVEL": "SYS_NOTICE",
			"METADATA_PROXY_SECRET": "contrail",
			"PHYSICAL_INTERFACE": "",
			"RABBITMQ_NODES": contrailNodes,
			"DNS_SERVER_PORT": "9053",
			"RABBITMQ_NODE_PORT": "5672",
			"REDIS_NODES": contrailNodes,
			"ZOOKEEPER_NODES": contrailNodes,
			"ANALYTICSDB_PORT": "9163",
			"ANALYTICSDB_CQL_PORT": "9045",
			"CONFIGDB_PORT": "9164",
			"CONFIGDB_CQL_PORT": "9044",
			"ANALYTICSDB_ENABLE": "true",
		},
	}
}

func newKubeManagerCm(cr *contrailoperatorsv1alpha1.InfraVars, apiServer string, apiServerPort string) *corev1.ConfigMap{
//	apiServer := cr.Spec.ApiServer
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:		"contrail-kubemanager-conf-env",
			Namespace:	cr.Namespace,
		},
		Data: map[string]string{
			"KUBERNETES_API_SERVER": apiServer,
			"KUBERNETES_API_SECURE_PORT": apiServerPort,
			"K8S_TOKEN_FILE": "/tmp/serviceaccount/token",
			"KUBERNETES_CLUSTER_NAME": "k8s",
			"KUBERNETES_CLUSTER_PROJECT": "{}",
			"KUBERNETES_CLUSTER_NETWORK": "{}",
			"KUBERNETES_POD_SUBNETS": "10.32.0.0/12",
			"KUBERNETES_IP_FABRIC_SUBNETS": "10.64.0.0/12",
			"KUBERNETES_SERVICE_SUBNETS": "10.96.0.0/12",
			"KUBERNETES_IP_FABRIC_FORWARDING": "false",
			"KUBERNETES_IP_FABRIC_SNAT": "true",
			"KUBERNETES_PUBLIC_FIP_POOL": "{}",
		},
	}
}

func newKubernetesCm(cr *contrailoperatorsv1alpha1.InfraVars, apiServer string, apiServerPort string) *corev1.ConfigMap{
//	apiServer := cr.Spec.ApiServer
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:		"contrail-kubernetes-conf-env",
			Namespace:	cr.Namespace,
		},
		Data: map[string]string{
			"KUBERNETES_API_SERVER": apiServer,
			"KUBERNETES_API_SECURE_PORT": apiServerPort,
			"K8S_TOKEN_FILE": "/tmp/serviceaccount/token",
		},
	}
}

func newAnalyticsZkCm(cr *contrailoperatorsv1alpha1.InfraVars) *corev1.ConfigMap{
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:		"contrail-analyticszk-conf-env",
			Namespace:	cr.Namespace,
		},
		Data: map[string]string{
			"ZOOKEEPER_PORT": "2181",
		  "ZOOKEEPER_PORTS": "2888:3888",
		},
	}
}

func newConfigZkCm(cr *contrailoperatorsv1alpha1.InfraVars) *corev1.ConfigMap{
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:		"contrail-configzk-conf-env",
			Namespace:	cr.Namespace,
		},
		Data: map[string]string{
			"ZOOKEEPER_PORT": "2181",
		  "ZOOKEEPER_PORTS": "2888:3888",
		},
	}
}

func newAnalyticsDbCm(cr *contrailoperatorsv1alpha1.InfraVars, contrailNodes string) *corev1.ConfigMap{
//	contrailNodes := cr.Spec.ContrailMasters
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:		"contrail-analyticsdb-conf-env",
			Namespace:	cr.Namespace,
		},
		Data: map[string]string{
			"CASSANDRA_SEEDS": contrailNodes,
		  "CASSANDRA_CLUSTER_NAME": "Contrail",
		  "CASSANDRA_START_RPC": "true",
		  "CASSANDRA_LISTEN_ADDRESS": "auto",
		  "CASSANDRA_PORT": "9163",
		  "CASSANDRA_CQL_PORT": "9045",
		  "CASSANDRA_SSL_STORAGE_PORT": "7004",
		  "CASSANDRA_STORAGE_PORT": "7003",
		  "CASSANDRA_JMX_LOCAL_PORT": "7203",
		  "ANALYTICSDB_PORT": "9163",
		  "ANALYTICSDB_CQL_PORT": "9045",
		  "CONFIGDB_PORT": "9164",
		  "CONFIGDB_CQL_PORT": "9044",
		},
	}
}

func newConfigDbCm(cr *contrailoperatorsv1alpha1.InfraVars, contrailNodes string) *corev1.ConfigMap{
//	contrailNodes := cr.Spec.ContrailMasters
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:		"contrail-configdb-conf-env",
			Namespace:	cr.Namespace,
		},
		Data: map[string]string{
			"CASSANDRA_SEEDS": contrailNodes,
		  "CASSANDRA_CLUSTER_NAME": "ContrailConfigDB",
		  "CASSANDRA_START_RPC": "true",
		  "CASSANDRA_LISTEN_ADDRESS": "auto",
		  "CASSANDRA_PORT": "9164",
		  "CASSANDRA_CQL_PORT": "9044",
		  "CASSANDRA_SSL_STORAGE_PORT": "7014",
		  "CASSANDRA_STORAGE_PORT": "7013",
		  "CASSANDRA_JMX_LOCAL_PORT": "7204",
		  "ANALYTICSDB_PORT": "9163",
		  "ANALYTICSDB_CQL_PORT": "9045",
		  "CONFIGDB_PORT": "9164",
		  "CONFIGDB_CQL_PORT": "9044",
		},
	}
}

func newRabbitCm(cr *contrailoperatorsv1alpha1.InfraVars) *corev1.ConfigMap{
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:		"contrail-rabbitmq-conf-env",
			Namespace:	cr.Namespace,
		},
		Data: map[string]string{
			"RABBITMQ_NODE_PORT": "5672",
			"RABBITMQ_ERLANG_COOKIE": "47EFF3BB-4786-46E0-A5BB-58455B3C2CB4",
		},
	}
}
