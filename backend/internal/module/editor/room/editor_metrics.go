package room

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	writerQueueGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "editor_writer_queue_length",
		Help: "Number of pending persist operations in the write-behind buffer.",
	})

	actorCmdQueueGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "editor_actor_cmd_queue_length",
		Help: "Number of pending commands in the map actor command channel.",
	}, []string{"map_code"})
)

func StartMetricsCollector(rm *RoomManager, w *Writer) {
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for range t.C {
			writerQueueGauge.Set(float64(w.QueueLen()))
			rm.mu.RLock()
			for mapCode, a := range rm.actors {
				actorCmdQueueGauge.WithLabelValues(mapCode).Set(float64(a.CmdQueueLen()))
			}
			rm.mu.RUnlock()
		}
	}()
}
