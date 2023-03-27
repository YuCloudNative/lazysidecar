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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

const (
	CsmEgressDefaultConfigName      = "default.conf"
	CsmEgressNginxConfigName        = "nginx.conf"
	CsmEgressStreamConfigName       = "stream.conf"
	CsmEgressWorkloadName           = "csm-egressgateway"
	CsmEgressDefaultConfigMountPath = "/etc/nginx/conf.d/default.conf"
	CsmEgressNginxConfigMountPath   = "/etc/nginx/nginx.conf"
	CsmEgressStreamConfigMountPath  = "/etc/nginx/conf.d/stream.conf"
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
				"app": CsmEgressWorkloadName,
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
		LocalObjectReference: corev1.LocalObjectReference{Name: CsmEgressWorkloadName},
		DefaultMode:          &defaultMode,
	}
	volume := corev1.Volume{
		Name:         CsmEgressWorkloadName,
		VolumeSource: corev1.VolumeSource{ConfigMap: configMapVolumeSource},
	}
	volumeList = append(volumeList, volume)
	volumeMount1 := corev1.VolumeMount{
		Name:      "default-conf",
		ReadOnly:  true,
		MountPath: CsmEgressDefaultConfigMountPath,
		SubPath:   CsmEgressDefaultConfigName,
	}
	volumeMount2 := corev1.VolumeMount{
		Name:      "nginx-conf",
		ReadOnly:  true,
		MountPath: CsmEgressNginxConfigMountPath,
		SubPath:   CsmEgressNginxConfigName,
	}
	volumeMount3 := corev1.VolumeMount{
		Name:      "stream-conf",
		ReadOnly:  true,
		MountPath: CsmEgressStreamConfigMountPath,
		SubPath:   CsmEgressStreamConfigName,
	}
	volumeMountList = append(volumeMountList, volumeMount1)
	volumeMountList = append(volumeMountList, volumeMount2)
	volumeMountList = append(volumeMountList, volumeMount3)
	container := corev1.Container{
		Name:            "openresty",
		Image:           csmEgressImage,
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    volumeMountList,
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
					"app": CsmEgressWorkloadName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                    CsmEgressWorkloadName,
						"csm.io/istio-injection": "enabled",
					},
				},
				Spec: corev1.PodSpec{
					Volumes:    volumeList,
					Containers: containerList,
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
	r.Create(ctx, defaultCsmEgressDeployment)
}
func (r *LazySidecarReconciler) CreateCsmEgressService(ctx context.Context) {
	defaultCsmEgressService := r.GetCsmEgressService()
	r.Create(ctx, defaultCsmEgressService)
}
func (r *LazySidecarReconciler) CreateCsmEgressConfigMap(ctx context.Context) {
	defaultCsmEgressConfigMap := r.GetCsmEgressConfigMap()
	r.Create(ctx, defaultCsmEgressConfigMap)
}
