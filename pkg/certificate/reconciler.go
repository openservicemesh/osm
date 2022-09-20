package certificate

import (
	"context"
	"time"

	"github.com/openservicemesh/osm/pkg/errcode"
)

// CheckAndUpdate checks for a MRC to reach a desired condition and when the condition is met,
// it updates the status and terminates the loop
func (r *MRCReconciler) CheckAndUpdate(ctx context.Context, mrcClient MRCClient, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				log.Info().Str(mrcShortName, r.mrcName).Msg("stopping status reconciliation loop")
				return
			case <-ticker.C:
				mrc := mrcClient.GetMeshRootCertificate(r.mrcName)
				if mrc == nil {
					log.Error().Str(mrcShortName, r.mrcName).Msg("failed to get MRC in status reconciliation loop")
					continue
				}

				meetsCondition, err := r.checkStatus(ctx, mrc)
				// checkStatus will return an error if the MRC is no longer in the expected rotation state
				// i.e. prior to this reconciliation loop, another component has already moved the rotation
				// to the next state after seeing the valid status condition without this check, the MRC
				// reconciliation loop might never terminate
				if err == ErrUnexpectedMRCStatusInReconciler {
					log.Info().Err(err).Str(mrcShortName, r.mrcName).Msg("exiting status reconciliation loop")
					return
				}
				if err == ErrMRCErrorStatusInReconciler {
					log.Info().Err(err).Str(mrcShortName, r.mrcName).Msg("waiting for non error state in reconciliation loop")
					continue
				}
				if err != nil {
					log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCheckingMRCStatus)).Str(mrcShortName, r.mrcName).
						Msg("failed to check the status of the MRC in the reconciliation loop")
					continue
				}

				if !meetsCondition {
					continue
				}

				// update status does not need to implement its own retry logic. Updating the status will
				// be retried in the MRC reconciler's next iteration
				if err := r.updateStatus(ctx, mrc); err != nil {
					log.Warn().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingMRCStatus)).Str(mrcShortName, r.mrcName).
						Msg("failed to update MRC status status reconciliation loop. Condition was met")
					continue
				}
				return
			}
		}
	}()
}
