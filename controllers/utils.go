package controllers

//import {
//corev1 "k8s.io/api/core/v1"
//}
//
//func podList() *corev1.PodList {
//
//}
//podList := &corev1.PodList{}
//listOpts := []client.ListOption{
//client.InNamespace(memcached.Namespace),
//client.MatchingLabels(map[string]string{"app": "memcached"}),
//client.MatchingFields{"status.phase": "Running"},
//}
//for _, item := range podList.Items {
//arrIPS = append(arrIPS, item.Status.PodIP+string(":11211"))
//}
