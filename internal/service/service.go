package service

import (
	"context"
	"math"

	"indeed.com/mjpitz/actuary/v1alpha"
)

func NewActuaryServer() v1alpha.ActuaryServiceServer {
	return &actuaryServer{}
}

type actuaryServer struct {
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func toBasisPoints(val float64) float64 {
	return math.Round(val * 10000)
}

func fromBasisPoints(val float64) float64 {
	return float64(val) / 10000
}

// Record current assumes all data provided in the request belongs to the same timestamp bucket.
// To do this properly, we need to tally by bucket of time.
// That shouldn't be too hard to do with a little coding.
func (a *actuaryServer) Record(ctx context.Context, req *v1alpha.RecordRequest) (*v1alpha.RecordResponse, error) {
	available := req.Available
	tally := make(map[string]uint64, len(available))

	for _, allocation := range req.Allocations {
		for key, value := range allocation.Detail {
			last, _ := tally[key]
			tally[key] = last + value
		}
	}

	for key, tallied := range tally {
		reported, _ := available[key]
		available[key] = max(reported, tallied)
	}

	lineItems := make([]*v1alpha.LineItem, len(req.Allocations))
	for i, allocation := range req.Allocations {
		usage := 0.0
		for key, max := range available {
			usage += toBasisPoints(float64(allocation.Detail[key]) / float64(max))
		}

		averageUsage := uint64(math.Round(usage / float64(len(available))))

		lineItems[i] = &v1alpha.LineItem{
			Datetime: allocation.Datetime,
			Payer:    allocation.Who,
			Payee:    ctx.Value("clientID").(string),
			Type:     v1alpha.LineItemType_DEBIT,
			Usage:    averageUsage,
			Urn:      allocation.What,
			Detail:   allocation.Detail,
			Labels:   allocation.Labels,
		}

		for key, value := range allocation.Detail {
			last, _ := tally[key]
			tally[key] = last + value
		}
	}

	// TODO: store line items when the DB is set up

	return &v1alpha.RecordResponse{}, nil
}

var _ v1alpha.ActuaryServiceServer = &actuaryServer{}
