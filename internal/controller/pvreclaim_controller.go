package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	reclaimv1alpha1 "github.com/greninja517/pv-reclaimer-controller/api/v1alpha1"
)

// PVReclaimReconciler reconciles a PVReclaim object
type PVReclaimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=reclaim.pv-reclaimer.io,resources=pvreclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=reclaim.pv-reclaimer.io,resources=pvreclaims/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=reclaim.pv-reclaimer.io,resources=pvreclaims/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;update;patch
func (r *PVReclaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// setting up the Logger
	log := r.Log.WithValues("pv-reclaim", req.Name)
	log.Info("Reconciliation Started")

	// fetching the actual object from informer cache
	pvReclaim := &reclaimv1alpha1.PVReclaim{}
	err := r.Client.Get(ctx, req.NamespacedName, pvReclaim)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "PVReclaim resource not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get PVReclaim")
		return ctrl.Result{}, err
	}

	// fetching the PV object
	pv := &corev1.PersistentVolume{}
	err = r.Client.Get(ctx, client.ObjectKey{Name: pvReclaim.Spec.PersistentVolumeName}, pv)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "PV resource doesn't exist")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get PV")
		return ctrl.Result{}, err
	}

	// check if the PV is in Released state
	if pv.Status.Phase != corev1.VolumeReleased {
		log.Info("pv not in Released state", "pv name", pv.Name, "current status", pv.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Check if the claimRef object is already  nil or not
	if pv.Spec.ClaimRef == nil {
		log.Info("PV not in Released state but no PVC is referenced in PV. Requeing...", "pv-name", pv.Name)
		return ctrl.Result{}, nil
	}

	// make the PV available for binding
	log.Info("Making the PV available for binding", "pv-name", pv.Name)
	pvCopied := pv.DeepCopy()
	pvCopied.Spec.ClaimRef = nil
	if err = r.Client.Update(ctx, pvCopied, &client.UpdateOptions{
		FieldManager: "pvreclaim-controller",
	}); err != nil {
		log.Error(err, "Failed to remove claimRef field from PV", "pv-name", pv.Name)
		return ctrl.Result{}, err
	}

	log.Info("PV is now available for binding. Reconciliation Successfull", "pv-name", pv.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PVReclaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&reclaimv1alpha1.PVReclaim{}).
		Named("pvreclaim").
		Complete(r)
}
