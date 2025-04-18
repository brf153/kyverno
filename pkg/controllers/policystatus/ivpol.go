package policystatus

import (
	"context"
	"fmt"

	policiesv1alpha1 "github.com/kyverno/kyverno/api/policies.kyverno.io/v1alpha1"
	celautogen "github.com/kyverno/kyverno/pkg/cel/autogen"
	engineapi "github.com/kyverno/kyverno/pkg/engine/api"
	controllerutils "github.com/kyverno/kyverno/pkg/utils/controller"
	datautils "github.com/kyverno/kyverno/pkg/utils/data"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c controller) updateIvpolStatus(ctx context.Context, ivpol *policiesv1alpha1.ImageValidatingPolicy) error {
	updateFunc := func(ivpol *policiesv1alpha1.ImageValidatingPolicy) error {
		p := engineapi.NewImageValidatingPolicy(ivpol)
		// conditions
		conditionStatus := c.reconcileConditions(ctx, p)
		ready := true
		for _, condition := range conditionStatus.Conditions {
			if condition.Status != metav1.ConditionTrue {
				ready = false
				break
			}
		}
		if conditionStatus.Ready == nil || conditionStatus.IsReady() != ready {
			conditionStatus.Ready = &ready
		}
		// autogen
		rules, err := celautogen.ImageValidatingPolicy(ivpol)
		if err != nil {
			return fmt.Errorf("failed to build autogen rules for ivpol %s: %v", ivpol.GetName(), err)
		}
		autogenStatus := policiesv1alpha1.ImageValidatingPolicyAutogenStatus{
			Configs: rules,
		}
		// assign
		ivpol.Status = policiesv1alpha1.ImageValidatingPolicyStatus{
			ConditionStatus: *conditionStatus,
			Autogen:         autogenStatus,
		}
		return nil
	}

	err := controllerutils.UpdateStatus(ctx,
		ivpol,
		c.client.PoliciesV1alpha1().ImageValidatingPolicies(),
		updateFunc,
		func(current, expect *policiesv1alpha1.ImageValidatingPolicy) bool {
			if current.GetStatus().GetConditionStatus().Ready == nil || current.GetStatus().GetConditionStatus().IsReady() != expect.GetStatus().GetConditionStatus().IsReady() {
				return false
			}

			if len(current.GetStatus().GetConditionStatus().Conditions) != len(expect.GetStatus().GetConditionStatus().Conditions) {
				return false
			}

			for _, condition := range current.GetStatus().GetConditionStatus().Conditions {
				for _, expectCondition := range expect.GetStatus().GetConditionStatus().Conditions {
					if condition.Type == expectCondition.Type && condition.Status != expectCondition.Status {
						return false
					}
				}
			}
			return datautils.DeepEqual(current.GetStatus().Autogen, expect.GetStatus().Autogen)
		},
	)
	return err
}
