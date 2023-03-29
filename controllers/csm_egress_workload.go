/**
* @Author: guohb65
* @Date: 2022-3-18 16:39
 */
package controllers

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/yucloudnative/lazysidecar/api/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

const (
	CsmEgressDefaultConfigName      = "default.conf"
	CsmEgressNginxConfigName        = "nginx.conf"
	CsmEgressStreamConfigName       = "stream.conf"
	DefaultMountName                = "config"
	CsmEgressDefaultConfigMountPath = "/etc/nginx/conf.d/default.conf"
	CsmEgressNginxConfigMountPath   = "/etc/nginx/nginx.conf"
	CsmEgressStreamConfigMountPath  = "/etc/nginx/stream.d/stream.conf"
)

func (r *LazySidecarReconciler) GetCsmEgressConfigMap() *corev1.ConfigMap {

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1.DefaultCsmEgressConfigmapName,
			Namespace: v1.ROOTNS,
		},
		Data: map[string]string{
			CsmEgressDefaultConfigName: csmEgressWorkloadDefaultConfigData,
			CsmEgressNginxConfigName:   csmEgressWorkloadNginxConfigData,
			CsmEgressStreamConfigName:  csmEgressWorkloadStreamConfigData,
		},
	}
	if err := controllerutil.SetControllerReference(ownerDeployment, configMap, r.Scheme); err != nil {
		logrus.Error("Set Reference: ", err)
	}
	return configMap
}
func (r *LazySidecarReconciler) GetCsmEgressService() *corev1.Service {
	var servicePortList []corev1.ServicePort
	if csmEgressWorkloadPort == "" {
		csmEgressWorkloadPort = "80"
	}
	csmEgressWorkloadPortInt, err := strconv.ParseInt(csmEgressWorkloadPort, 10, 32)
	if err != nil {
		logrus.Error("csm egress workload port conversion failed: ", err)
	}
	targetPort := intstr.IntOrString{
		IntVal: int32(csmEgressWorkloadPortInt),
	}
	servicePort := corev1.ServicePort{
		Name:       "http",
		Protocol:   corev1.ProtocolTCP,
		Port:       80,
		TargetPort: targetPort,
	}
	servicePortList = append(servicePortList, servicePort)
	serviceInstance := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1.DefaultCsmEgressServiceName,
			Namespace: v1.ROOTNS,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": v1.DefaultCsmEgressDeploymentName,
			},
			Ports: servicePortList,
		},
	}
	if err := controllerutil.SetControllerReference(ownerDeployment, serviceInstance, r.Scheme); err != nil {
		logrus.Error("Set Reference: ", err)
	}
	return serviceInstance
}

func (r *LazySidecarReconciler) GetCsmEgressDeployment() *appv1.Deployment {
	var (
		defaultMode     int32 = 420
		volumeList      []corev1.Volume
		keyToPathList   []corev1.KeyToPath
		containerList   []corev1.Container
		volumeMountList []corev1.VolumeMount
		envList         []corev1.EnvVar
	)
	keyToPath1 := corev1.KeyToPath{
		Key:  CsmEgressDefaultConfigName,
		Path: CsmEgressDefaultConfigName,
	}
	keyToPath2 := corev1.KeyToPath{
		Key:  CsmEgressNginxConfigName,
		Path: CsmEgressNginxConfigName,
	}
	keyToPath3 := corev1.KeyToPath{
		Key:  CsmEgressStreamConfigName,
		Path: CsmEgressStreamConfigName,
	}
	keyToPathList = append(keyToPathList, keyToPath1)
	keyToPathList = append(keyToPathList, keyToPath2)
	keyToPathList = append(keyToPathList, keyToPath3)
	configMapVolumeSource := &corev1.ConfigMapVolumeSource{
		Items:                keyToPathList,
		LocalObjectReference: corev1.LocalObjectReference{Name: v1.DefaultCsmEgressConfigmapName},
		DefaultMode:          &defaultMode,
	}
	volume := corev1.Volume{
		Name:         DefaultMountName,
		VolumeSource: corev1.VolumeSource{ConfigMap: configMapVolumeSource},
	}
	volumeList = append(volumeList, volume)
	volumeMount1 := corev1.VolumeMount{
		Name:      DefaultMountName,
		ReadOnly:  true,
		MountPath: CsmEgressDefaultConfigMountPath,
		SubPath:   CsmEgressDefaultConfigName,
	}
	volumeMount2 := corev1.VolumeMount{
		Name:      DefaultMountName,
		ReadOnly:  true,
		MountPath: CsmEgressNginxConfigMountPath,
		SubPath:   CsmEgressNginxConfigName,
	}
	volumeMount3 := corev1.VolumeMount{
		Name:      DefaultMountName,
		ReadOnly:  true,
		MountPath: CsmEgressStreamConfigMountPath,
		SubPath:   CsmEgressStreamConfigName,
	}
	volumeMountList = append(volumeMountList, volumeMount1)
	volumeMountList = append(volumeMountList, volumeMount2)
	volumeMountList = append(volumeMountList, volumeMount3)
	env := corev1.EnvVar{
		Name: "POD_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	}
	envList = append(envList, env)
	container := corev1.Container{
		Name:            v1.DefaultCsmEgressServiceName,
		Image:           csmEgressImage,
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    volumeMountList,
		Env:             envList,
		//Resources:       r.GetResources(),
	}
	containerList = append(containerList, container)
	deploymentInstance := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1.DefaultCsmEgressDeploymentName,
			Namespace: v1.ROOTNS,
		},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": v1.DefaultCsmEgressDeploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                    v1.DefaultCsmEgressDeploymentName,
						"csm.io/istio-injection": "enabled",
					},
				},
				Spec: corev1.PodSpec{
					Volumes:            volumeList,
					Containers:         containerList,
					ServiceAccountName: v1.DefaultCsmEgressDeploymentName,
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(ownerDeployment, deploymentInstance, r.Scheme); err != nil {
		logrus.Error("Set Reference: ", err)
	}
	return deploymentInstance
}

func (r *LazySidecarReconciler) CreateCsmEgressDeployment(ctx context.Context) {
	defaultCsmEgressDeployment := r.GetCsmEgressDeployment()
	err := r.Create(ctx, defaultCsmEgressDeployment)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		logrus.Error(err)
	}
}
func (r *LazySidecarReconciler) CreateCsmEgressService(ctx context.Context) {
	defaultCsmEgressService := r.GetCsmEgressService()
	err := r.Create(ctx, defaultCsmEgressService)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		logrus.Error(err)
	}
}
func (r *LazySidecarReconciler) CreateCsmEgressConfigMap(ctx context.Context) {
	defaultCsmEgressConfigMap := r.GetCsmEgressConfigMap()
	err := r.Create(ctx, defaultCsmEgressConfigMap)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		logrus.Error(err)
	}
}
func (r *LazySidecarReconciler) CreateCsmSa(ctx context.Context) {
	defaultGlobalSidecarSa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1.DefaultCsmEgressDeploymentName,
			Namespace: v1.ROOTNS,
		},
	}
	if ownerDeployment.Name == "" {
		r.Init()
	}
	if err := controllerutil.SetControllerReference(ownerDeployment, defaultGlobalSidecarSa, r.Scheme); err != nil {
		logrus.Error("Set Reference: ", err)
	}
	err := r.Create(ctx, defaultGlobalSidecarSa)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		logrus.Error("CreateDefaultSA err: ", err)
	}
}
