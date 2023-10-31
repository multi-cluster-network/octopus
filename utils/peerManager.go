package utils

import (
	"context"

	"github.com/multi-cluster-network/octopus/pkg/apis/octopus.io/v1alpha1"
	clientset "github.com/multi-cluster-network/octopus/pkg/generated/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

// ApplyEndPointSliceWithRetry create or update existed slices.
func ApplyPeerWithRetry(client clientset.Interface, peer *v1alpha1.Peer) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		var lastError error
		_, lastError = client.OctopusV1alpha1().Peers(peer.GetNamespace()).Create(context.TODO(), peer, metav1.CreateOptions{})
		if lastError == nil {
			return nil
		}
		if !errors.IsAlreadyExists(lastError) {
			return lastError
		}

		curObj, err := client.OctopusV1alpha1().Peers(peer.GetNamespace()).Get(context.TODO(), peer.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}
		lastError = nil

		if ResourceNeedResync(curObj, peer, false) {
			// try to update peer
			curObj.Spec.PodCIDR = peer.Spec.PodCIDR
			curObj.Spec.Endpoint = peer.Spec.Endpoint
			curObj.Spec.PublicKey = peer.Spec.PublicKey
			curObj.Spec.ClusterID = peer.Spec.ClusterID
			_, lastError = client.OctopusV1alpha1().Peers(peer.GetNamespace()).Update(context.TODO(), curObj, metav1.UpdateOptions{})
			if lastError == nil {
				return nil
			}
		}
		return lastError
	})
}

func DeletePeerWithRetry(client clientset.Interface, name, namespace string) error {
	var err error
	err = wait.ExponentialBackoffWithContext(context.TODO(), retry.DefaultBackoff, func(ctx context.Context) (bool, error) {
		if err = client.OctopusV1alpha1().Peers(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
			return false, err
		}

		if err == nil || (err != nil && errors.IsNotFound(err)) {
			return true, nil
		}
		return false, nil
	})
	if err == nil {
		return nil
	}
	return err
}