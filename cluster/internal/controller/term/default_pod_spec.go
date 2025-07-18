package term

import (
	corev1 "k8s.io/api/core/v1"
)

var podSpec = &corev1.PodSpec{
	Volumes: []corev1.Volume{},
	Containers: []corev1.Container{{
		Name:    "cloudterm",
		Command: []string{"/bin/sh", "-c", "sh k8s-init.sh"},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 5555,
				Protocol:      "TCP",
			},
			{
				ContainerPort: 8888,
				Protocol:      "TCP",
			},
		},
		EnvFrom: []corev1.EnvFromSource{},
		Env: []corev1.EnvVar{
			{
				Name:  "PATH",
				Value: "/system/bin:/system/xbin",
			},
		},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"SETPCAP",
					"AUDIT_WRITE",
					"SYS_CHROOT",
					"CHOWN",
					"DAC_OVERRIDE",
					"FOWNER",
					"SETGID",
					"SETUID",
					"SYSLOG",
					"SYS_ADMIN",
					"WAKE_ALARM",
					"SYS_PTRACE",
					"BLOCK_SUSPEND",
					"MKNOD",
					"KILL",
					"NET_RAW",
					"NET_BIND_SERVICE",
					"NET_ADMIN",
					"SYS_NICE",
					"AUDIT_CONTROL",
					"DAC_READ_SEARCH",
					"IPC_LOCK",
					"SYS_MODULE",
					"SYS_RESOURCE",
				},
				Drop: []corev1.Capability{
					"ALL",
				},
			},
			RunAsUser:  new(int64),
			RunAsGroup: new(int64),
		},
	}},
}
