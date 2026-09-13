package persistence

import (
	"testing"

	"loyalty-nexus/internal/domain/entities"
)

func TestSortRouteCandidatesFreeFirst(t *testing.T) {
	rows := []entities.AIRouteCandidate{
		{Binding: entities.AIToolProviderBinding{Priority: 1, CostTier: entities.CostTierPremium}},
		{Binding: entities.AIToolProviderBinding{Priority: 3, CostTier: entities.CostTierFree}},
		{Binding: entities.AIToolProviderBinding{Priority: 2, CostTier: entities.CostTierFree}},
		{Binding: entities.AIToolProviderBinding{Priority: 1, CostTier: entities.CostTierLowCost}},
	}
	sortRouteCandidates(entities.RoutingFreeFirst, rows)
	got := []string{rows[0].Binding.CostTier, rows[1].Binding.CostTier, rows[2].Binding.CostTier, rows[3].Binding.CostTier}
	want := []string{entities.CostTierFree, entities.CostTierFree, entities.CostTierLowCost, entities.CostTierPremium}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %s want %s", i, got[i], want[i])
		}
	}
	if rows[0].Binding.Priority != 2 {
		t.Fatalf("free candidates not ordered by priority")
	}
}

func TestSortRouteCandidatesQualityFirstHonoursPriority(t *testing.T) {
	rows := []entities.AIRouteCandidate{
		{Binding: entities.AIToolProviderBinding{Priority: 2, CostTier: entities.CostTierFree}},
		{Binding: entities.AIToolProviderBinding{Priority: 1, CostTier: entities.CostTierPremium}},
	}
	sortRouteCandidates(entities.RoutingQualityFirst, rows)
	if rows[0].Binding.Priority != 1 {
		t.Fatalf("quality-first must honour admin priority")
	}
}
