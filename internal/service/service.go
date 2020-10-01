package service

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"gorm.io/datatypes"

	"gorm.io/gorm"

	"indeed.com/mjpitz/actuary/internal/db"
	"indeed.com/mjpitz/actuary/v1alpha"
)

func NewActuaryServer(db *gorm.DB) v1alpha.ActuaryServiceServer {
	return &actuaryServer{
		db: db,
	}
}

type actuaryServer struct {
	db *gorm.DB
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

	lineItems := make([]*db.LineItem, len(req.Allocations))
	for i, allocation := range req.Allocations {
		usage := 0.0
		for key, max := range available {
			usage += toBasisPoints(float64(allocation.Detail[key]) / float64(max))
		}

		averageUsage := int64(math.Round(usage / float64(len(available))))

		detailJSON, _ := json.Marshal(allocation.Detail)
		labelsJSON, _ := json.Marshal(allocation.Labels)

		lineItems[i] = &db.LineItem{
			DateTime: time.Unix(allocation.Datetime.Seconds, int64(allocation.Datetime.Nanos)),
			Payer:    allocation.Who,
			//Payee:    ctx.Value("clientID").(string),
			Payee:  "",
			Kind:   db.DebitLineItemKind,
			Usage:  averageUsage,
			URN:    allocation.What,
			Detail: datatypes.JSON(detailJSON),
			Labels: datatypes.JSON(labelsJSON),
		}
	}

	tx := a.db.Create(lineItems)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &v1alpha.RecordResponse{}, nil
}

var _ v1alpha.ActuaryServiceServer = &actuaryServer{}
