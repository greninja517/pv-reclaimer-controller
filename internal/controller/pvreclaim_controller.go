package controller

import (
	"context"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	var reconcileErr error = nil
	// setting up the Logger
	log := r.Log.WithValues("pv-reclaim", req.NamespacedName)
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
		reconcileErr = err
		return ctrl.Result{}, reconcileErr
	}

	// check if the PVReclaim object should be reconciled or not
	if pvReclaim.Generation == pvReclaim.Status.ObservedGeneration && (pvReclaim.Status.Phase == reclaimv1alpha1.SuccessPhase || pvReclaim.Status.Phase == reclaimv1alpha1.FailurePhase) {
		log.Info("PVReclaim object already processed. Skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// updating the status only if it has actually changed in this code
	originalStatus := pvReclaim.Status.DeepCopy()
	defer func() {
		if !reflect.DeepEqual(originalStatus, &pvReclaim.Status) {
			// change took place. so, updating
			log.Info("Updating the PVReclaim object Status")
			if pvReclaim.Status.Phase == reclaimv1alpha1.FailurePhase || pvReclaim.Status.Phase == reclaimv1alpha1.SuccessPhase {
				pvReclaim.Status.ObservedGeneration = pvReclaim.Generation
			}
			if err := r.Status().Update(ctx, pvReclaim); err != nil {
				log.Error(err, "Failed to update the Status of PVReclaim object")
				if reconcileErr == nil {
					reconcileErr = err
				}
			}
		}
	}()

	if pvReclaim.Status.Phase == "" {
		// settting the initial phase
		pvReclaim.Status.Phase = reclaimv1alpha1.PendingPhase
	}

	// fetching the PV object
	pv := &corev1.PersistentVolume{}
	err = r.Get(ctx, client.ObjectKey{Name: pvReclaim.Spec.PersistentVolumeName}, pv)
	if err != nil {
		pvReclaim.Status.Phase = reclaimv1alpha1.FailurePhase
		if apierrors.IsNotFound(err) {
			log.Error(err, "PV resource doesn't exist", "pvName", pvReclaim.Spec.PersistentVolumeName)
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get PV", "pvName", pvReclaim.Spec.PersistentVolumeName)
		reconcileErr = err
		return ctrl.Result{}, reconcileErr
	}

	// check if the PV is in Released state
	if pv.Status.Phase != corev1.VolumeReleased {
		log.Info("pv not in Released state", "pv name", pv.Name, "current status", pv.Status.Phase)
		pvReclaim.Status.Phase = reclaimv1alpha1.FailurePhase
		return ctrl.Result{}, nil
	}

	// Check if the claimRef object is already  nil or not
	if pv.Spec.ClaimRef == nil {
		log.Info("PV already in Available state OR API server is updating state of PV since no PVC is referenced in PV.", "pv-name", pv.Name)
		pvReclaim.Status.Phase = reclaimv1alpha1.SuccessPhase
		pvReclaim.Status.ReclaimedTimestamp = &metav1.Time{Time: time.Now()}
		return ctrl.Result{}, reconcileErr
	}

	// make the PV available for binding
	log.Info("Making the PV available for binding", "pv-name", pv.Name)
	pvCopied := pv.DeepCopy()
	pvCopied.Spec.ClaimRef = nil
	if err = r.Client.Update(ctx, pvCopied, &client.UpdateOptions{
		FieldManager: "pvreclaim-controller",
	}); err != nil {
		log.Error(err, "Failed to remove claimRef field from PV. Requeuing ...", "pv-name", pv.Name)
		reconcileErr = err
		return ctrl.Result{}, reconcileErr
	}

	log.Info("PV is now available for binding. Reconciliation Successfull", "pv-name", pv.Name)
	pvReclaim.Status.Phase = reclaimv1alpha1.SuccessPhase
	pvReclaim.Status.ReclaimedTimestamp = &metav1.Time{Time: time.Now()}

	return ctrl.Result{}, reconcileErr
}

// SetupWithManager sets up the controller with the Manager.
func (r *PVReclaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&reclaimv1alpha1.PVReclaim{}).
		Named("pvreclaim").
		Complete(r)
}
