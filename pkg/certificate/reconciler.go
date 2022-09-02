package certificate

import (
	"context"
	"time"
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
				log.Info().Msgf("Stopping status reconciliation loop for MRC %s", r.mrcName)
				return
			case <-ticker.C:
				mrc := mrcClient.Get(r.mrcName)
				if mrc == nil {
					log.Error().Msgf("failed to get MRC %s in status reconciliation loop", r.mrcName)
					continue
				}

				meetsCondition, err := r.checkStatus(ctx, mrc)
				// checkStatus will return an error if the MRC is no longer in the expected rotation state
				// i.e. prior to this reconciliation loop, another component has already moved the rotation
				// to the next state after seeing the valid status condition without this check, the MRC
				// reconciliation loop might never terminate
				if err != nil {
					return
				}
				if !meetsCondition {
					continue
				}

				// update status does not need to implement its own retry logic. Updating the status will
				// be retried in the MRC reconciler's next iteration
				if err := r.updateStatus(ctx, mrc); err != nil {
					log.Warn().Err(err).Msgf("failed to update status of MRC %s in status reconciliation loop. Condition was met", r.mrcName)
					continue
				}
				return
			}
		}
	}()
}
