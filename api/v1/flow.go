package v1

import (
	"github.com/fs714/goiftop/accounting"
	"github.com/fs714/goiftop/utils/config"
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetFlows(c *gin.Context) {
	// Similar to how PrintNotifier works, we aggregate flows over a duration.
	// We can use the print interval as a reasonable default for the web UI.
	duration := config.PrintInterval
	if duration <= 0 {
		duration = 2
	}

	// A map to hold aggregated flows from all interfaces
	aggregatedFlows := make(map[accounting.FlowFingerprint]*accounting.Flow)

	for _, flowColHist := range accounting.GlobalAcct.FlowAccd {
		fc, _ := flowColHist.AggregationByDuration(duration)

		var flowMap map[accounting.FlowFingerprint]*accounting.Flow
		if config.IsDecodeL4 {
			flowMap = fc.L4FlowMap
		} else {
			flowMap = fc.L3FlowMap
		}

		for fp, f := range flowMap {
			if existingFlow, ok := aggregatedFlows[fp]; !ok {
				newFlow := accounting.FlowPool.Get().(*accounting.Flow)
				*newFlow = *f
				aggregatedFlows[fp] = newFlow
			} else {
				existingFlow.InboundBytes += f.InboundBytes
				existingFlow.InboundPackets += f.InboundPackets
				existingFlow.OutboundBytes += f.OutboundBytes
				existingFlow.OutboundPackets += f.OutboundPackets
			}
		}
	}

	// Convert map to slice for JSON response
	flowSlice := make([]*accounting.Flow, 0, len(aggregatedFlows))
	for _, f := range aggregatedFlows {
		flowSlice = append(flowSlice, f)
	}

	// Return flows and clean up the pool
	c.JSON(http.StatusOK, flowSlice)
	for _, f := range aggregatedFlows {
		accounting.FlowPool.Put(f)
	}
}
