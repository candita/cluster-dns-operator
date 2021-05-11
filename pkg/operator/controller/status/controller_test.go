package status

import (
	operatorconfig "github.com/openshift/cluster-dns-operator/pkg/operator/config"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestComputeOperatorProgressingCondition(t *testing.T) {
	type versions struct {
		operator, operand string
	}

	testCases := []struct {
		description       string
		dnsMissing        bool
		dnsAvailable      bool
		dnsProgressing    bool
		reportedVersions  versions
		oldVersions       versions
		curVersions       versions
		expectProgressing configv1.ConditionStatus
	}{
		{
			description:       "dns does not exist",
			dnsMissing:        true,
			expectProgressing: configv1.ConditionTrue,
		},
		{
			description:       "dns not available, not progressing",
			expectProgressing: configv1.ConditionFalse,
		},
		{
			description:       "dns not available but progressing",
			dnsProgressing:    true,
			expectProgressing: configv1.ConditionTrue,
		},
		{
			description:       "dns available, not progressing",
			dnsAvailable:      true,
			expectProgressing: configv1.ConditionFalse,
		},
		{
			description:       "dns available and progressing",
			dnsAvailable:      true,
			dnsProgressing:    true,
			expectProgressing: configv1.ConditionTrue,
		},
		{
			description:       "versions match",
			dnsAvailable:      true,
			reportedVersions:  versions{"v1", "dns-v1"},
			oldVersions:       versions{"v1", "dns-v1"},
			curVersions:       versions{"v1", "dns-v1"},
			expectProgressing: configv1.ConditionFalse,
		},
		{
			description:       "operator upgrade in progress",
			dnsAvailable:      true,
			reportedVersions:  versions{"v1", "dns-v1"},
			oldVersions:       versions{"v1", "dns-v1"},
			curVersions:       versions{"v2", "dns-v1"},
			expectProgressing: configv1.ConditionTrue,
		},
		{
			description:       "operand upgrade in progress",
			dnsAvailable:      true,
			reportedVersions:  versions{"v1", "dns-v1"},
			oldVersions:       versions{"v1", "dns-v1"},
			curVersions:       versions{"v1", "dns-v2"},
			expectProgressing: configv1.ConditionTrue,
		},
		{
			description:       "operator and operand upgrade in progress",
			dnsAvailable:      true,
			reportedVersions:  versions{"v1", "dns-v1"},
			oldVersions:       versions{"v1", "dns-v1"},
			curVersions:       versions{"v2", "dns-v2"},
			expectProgressing: configv1.ConditionTrue,
		},
		{
			description:       "operator upgrade done",
			dnsAvailable:      true,
			reportedVersions:  versions{"v2", "dns-v1"},
			oldVersions:       versions{"v1", "dns-v1"},
			curVersions:       versions{"v2", "dns-v1"},
			expectProgressing: configv1.ConditionFalse,
		},
		{
			description:       "operand upgrade done",
			dnsAvailable:      true,
			reportedVersions:  versions{"v1", "dns-v2"},
			oldVersions:       versions{"v1", "dns-v1"},
			curVersions:       versions{"v1", "dns-v2"},
			expectProgressing: configv1.ConditionFalse,
		},
		{
			description:       "operator and operand upgrade done",
			dnsAvailable:      true,
			reportedVersions:  versions{"v2", "dns-v2"},
			oldVersions:       versions{"v1", "dns-v1"},
			curVersions:       versions{"v2", "dns-v2"},
			expectProgressing: configv1.ConditionFalse,
		},
		{
			description:       "operator upgrade in progress, operand upgrade done",
			dnsAvailable:      true,
			reportedVersions:  versions{"v2", "dns-v1"},
			oldVersions:       versions{"v1", "dns-v1"},
			curVersions:       versions{"v2", "dns-v2"},
			expectProgressing: configv1.ConditionTrue,
		},
	}

	for _, tc := range testCases {
		var (
			haveDNS bool
			dns     *operatorv1.DNS
		)
		if !tc.dnsMissing {
			haveDNS = true
			availableStatus := operatorv1.ConditionFalse
			if tc.dnsAvailable {
				availableStatus = operatorv1.ConditionTrue
			}
			progressingStatus := operatorv1.ConditionFalse
			if tc.dnsProgressing {
				progressingStatus = operatorv1.ConditionTrue
			}
			dns = &operatorv1.DNS{
				Status: operatorv1.DNSStatus{
					Conditions: []operatorv1.OperatorCondition{{
						Type:   operatorv1.OperatorStatusTypeAvailable,
						Status: availableStatus,
					}, {
						Type:   operatorv1.OperatorStatusTypeProgressing,
						Status: progressingStatus,
					}},
				},
			}
		}
		oldVersions := []configv1.OperandVersion{
			{
				Name:    OperatorVersionName,
				Version: tc.oldVersions.operator,
			},
			{
				Name:    CoreDNSVersionName,
				Version: tc.oldVersions.operand,
			},
			{
				Name:    OpenshiftCLIVersionName,
				Version: tc.oldVersions.operand,
			},
			{
				Name:    KubeRBACProxyName,
				Version: tc.oldVersions.operand,
			},
		}
		reportedVersions := []configv1.OperandVersion{
			{
				Name:    OperatorVersionName,
				Version: tc.reportedVersions.operator,
			},
			{
				Name:    CoreDNSVersionName,
				Version: tc.reportedVersions.operand,
			},
			{
				Name:    OpenshiftCLIVersionName,
				Version: tc.reportedVersions.operand,
			},
			{
				Name:    KubeRBACProxyName,
				Version: tc.reportedVersions.operand,
			},
		}

		expected := configv1.ClusterOperatorStatusCondition{
			Type:   configv1.OperatorProgressing,
			Status: tc.expectProgressing,
		}

		actual := computeOperatorProgressingCondition(haveDNS, dns, oldVersions, reportedVersions,
			tc.curVersions.operator, tc.curVersions.operand, tc.curVersions.operand, tc.curVersions.operand)
		conditionsCmpOpts := []cmp.Option{
			cmpopts.IgnoreFields(configv1.ClusterOperatorStatusCondition{}, "LastTransitionTime", "Reason", "Message"),
		}
		if !cmp.Equal(actual, expected, conditionsCmpOpts...) {
			t.Errorf("%q: expected %#v, got %#v", tc.description, expected, actual)
		}
	}
}

func TestOperatorStatusesEqual(t *testing.T) {
	testCases := []struct {
		description string
		expected    bool
		a, b        configv1.ClusterOperatorStatus
	}{
		{
			description: "zero-valued ClusterOperatorStatus should be equal",
			expected:    true,
		},
		{
			description: "nil and non-nil slices are equal",
			expected:    true,
			a: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{},
			},
		},
		{
			description: "empty slices should be equal",
			expected:    true,
			a: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{},
			},
			b: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{},
			},
		},
		{
			description: "check no change in versions",
			expected:    true,
			a: configv1.ClusterOperatorStatus{
				Versions: []configv1.OperandVersion{
					{
						Name:    "operator",
						Version: "v1",
					},
					{
						Name:    "router",
						Version: "v2",
					},
				},
			},
			b: configv1.ClusterOperatorStatus{
				Versions: []configv1.OperandVersion{
					{
						Name:    "operator",
						Version: "v1",
					},
					{
						Name:    "router",
						Version: "v2",
					},
				},
			},
		},
		{
			description: "condition LastTransitionTime should not be ignored",
			expected:    false,
			a: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:               configv1.OperatorAvailable,
						Status:             configv1.ConditionTrue,
						LastTransitionTime: metav1.Unix(0, 0),
					},
				},
			},
			b: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:               configv1.OperatorAvailable,
						Status:             configv1.ConditionTrue,
						LastTransitionTime: metav1.Unix(1, 0),
					},
				},
			},
		},
		{
			description: "order of versions should not matter",
			expected:    true,
			a: configv1.ClusterOperatorStatus{
				Versions: []configv1.OperandVersion{
					{
						Name:    "operator",
						Version: "v1",
					},
					{
						Name:    "router",
						Version: "v2",
					},
				},
			},
			b: configv1.ClusterOperatorStatus{
				Versions: []configv1.OperandVersion{
					{
						Name:    "router",
						Version: "v2",
					},
					{
						Name:    "operator",
						Version: "v1",
					},
				},
			},
		},
		{
			description: "check missing related objects",
			expected:    false,
			a: configv1.ClusterOperatorStatus{
				RelatedObjects: []configv1.ObjectReference{
					{
						Name: "openshift-ingress",
					},
					{
						Name: "default",
					},
				},
			},
			b: configv1.ClusterOperatorStatus{
				RelatedObjects: []configv1.ObjectReference{
					{
						Name: "default",
					},
				},
			},
		},
		{
			description: "check extra related objects",
			expected:    false,
			a: configv1.ClusterOperatorStatus{
				RelatedObjects: []configv1.ObjectReference{
					{
						Name: "default",
					},
				},
			},
			b: configv1.ClusterOperatorStatus{
				RelatedObjects: []configv1.ObjectReference{
					{
						Name: "openshift-ingress",
					},
					{
						Name: "default",
					},
				},
			},
		},
		{
			description: "check condition reason differs",
			expected:    false,
			a: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionFalse,
						Reason: "foo",
					},
				},
			},
			b: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: configv1.ConditionFalse,
						Reason: "bar",
					},
				},
			},
		},
		{
			description: "check duplicate with single condition",
			expected:    false,
			a: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:    configv1.OperatorAvailable,
						Message: "foo",
					},
				},
			},
			b: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:    configv1.OperatorAvailable,
						Message: "foo",
					},
					{
						Type:    configv1.OperatorAvailable,
						Message: "foo",
					},
				},
			},
		},
		{
			description: "check duplicate with multiple conditions",
			expected:    false,
			a: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type: configv1.OperatorAvailable,
					},
					{
						Type: configv1.OperatorProgressing,
					},
					{
						Type: configv1.OperatorAvailable,
					},
				},
			},
			b: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type: configv1.OperatorProgressing,
					},
					{
						Type: configv1.OperatorAvailable,
					},
					{
						Type: configv1.OperatorProgressing,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		if actual := operatorStatusesEqual(tc.a, tc.b); actual != tc.expected {
			t.Errorf("%q: expected %v, got %v", tc.description, tc.expected, actual)
		}
	}
}

func TestComputeOperatorStatusVersions(t *testing.T) {
	type versions struct {
		operator string
		operand  string
	}

	testCases := []struct {
		description      string
		oldVersions      versions
		curVersions      versions
		progressing      configv1.ConditionStatus
		expectedVersions versions
	}{
		{
			description:      "initialize versions, operator is not progressing",
			oldVersions:      versions{UnknownVersionValue, UnknownVersionValue},
			curVersions:      versions{"v1", "dns-v1"},
			progressing:      configv1.ConditionFalse,
			expectedVersions: versions{"v1", "dns-v1"},
		},
		{
			description:      "initialize versions, progressing status unknown",
			oldVersions:      versions{UnknownVersionValue, UnknownVersionValue},
			curVersions:      versions{"v1", "dns-v1"},
			progressing:      configv1.ConditionUnknown,
			expectedVersions: versions{UnknownVersionValue, UnknownVersionValue},
		},
		{
			description:      "initialize versions, operator is progressing",
			oldVersions:      versions{UnknownVersionValue, UnknownVersionValue},
			curVersions:      versions{"v1", "dns-v1"},
			progressing:      configv1.ConditionTrue,
			expectedVersions: versions{UnknownVersionValue, UnknownVersionValue},
		},
		{
			description:      "update with no change",
			oldVersions:      versions{"v1", "dns-v1"},
			curVersions:      versions{"v1", "dns-v1"},
			progressing:      configv1.ConditionFalse,
			expectedVersions: versions{"v1", "dns-v1"},
		},
		{
			description:      "update operator version, operator is progressing",
			oldVersions:      versions{"v1", "dns-v1"},
			curVersions:      versions{"v2", "dns-v1"},
			progressing:      configv1.ConditionTrue,
			expectedVersions: versions{"v1", "dns-v1"},
		},
		{
			description:      "update operator version, operator is not progressing",
			oldVersions:      versions{"v1", "dns-v1"},
			curVersions:      versions{"v2", "dns-v1"},
			progressing:      configv1.ConditionFalse,
			expectedVersions: versions{"v2", "dns-v1"},
		},
		{
			description:      "update operand image, operator is progressing",
			oldVersions:      versions{"v1", "dns-v1"},
			curVersions:      versions{"v1", "dns-v2"},
			progressing:      configv1.ConditionTrue,
			expectedVersions: versions{"v1", "dns-v1"},
		},
		{
			description:      "update operand image, operator is not progressing",
			oldVersions:      versions{"v1", "dns-v1"},
			curVersions:      versions{"v1", "dns-v2"},
			progressing:      configv1.ConditionFalse,
			expectedVersions: versions{"v1", "dns-v2"},
		},
		{
			description: "update operator and operand image, operator is progressing",
			oldVersions: versions{"v1", "dns-v1"},
			curVersions: versions{"v2", "dns-v2"},
			progressing: configv1.ConditionTrue,

			expectedVersions: versions{"v1", "dns-v1"},
		},
		{
			description:      "update operator and operand image, operator is not progressing",
			oldVersions:      versions{"v1", "dns-v1"},
			curVersions:      versions{"v2", "dns-v2"},
			progressing:      configv1.ConditionFalse,
			expectedVersions: versions{"v2", "dns-v2"},
		},
	}

	for _, tc := range testCases {
		var (
			oldVersions      []configv1.OperandVersion
			newVersions      []configv1.OperandVersion
			expectedVersions []configv1.OperandVersion
		)

		oldVersions = []configv1.OperandVersion{
			{
				Name:    OperatorVersionName,
				Version: tc.oldVersions.operator,
			},
			{
				Name:    CoreDNSVersionName,
				Version: tc.oldVersions.operand,
			},
			{
				Name:    OpenshiftCLIVersionName,
				Version: tc.oldVersions.operand,
			},
			{
				Name:    KubeRBACProxyName,
				Version: tc.oldVersions.operand,
			},
		}
		newVersions = []configv1.OperandVersion{
			{
				Name:    OperatorVersionName,
				Version: tc.curVersions.operator,
			},
			{
				Name:    CoreDNSVersionName,
				Version: tc.curVersions.operand,
			},
			{
				Name:    OpenshiftCLIVersionName,
				Version: tc.curVersions.operand,
			},
			{
				Name:    KubeRBACProxyName,
				Version: tc.curVersions.operand,
			},
		}
		expectedVersions = []configv1.OperandVersion{
			{
				Name:    OperatorVersionName,
				Version: tc.expectedVersions.operator,
			},
			{
				Name:    CoreDNSVersionName,
				Version: tc.expectedVersions.operand,
			},
			{
				Name:    OpenshiftCLIVersionName,
				Version: tc.expectedVersions.operand,
			},
			{
				Name:    KubeRBACProxyName,
				Version: tc.expectedVersions.operand,
			},
		}

		operatorProgressingCondition := configv1.ClusterOperatorStatusCondition{
			Status: tc.progressing,
			Type:   configv1.OperatorProgressing,
		}

		r := &reconciler{
			Config: operatorconfig.Config{
				OperatorReleaseVersion: tc.curVersions.operator,
				CoreDNSImage:           tc.curVersions.operand,
				OpenshiftCLIImage:      tc.curVersions.operand,
				KubeRBACProxyImage:     tc.curVersions.operand,
			},
		}
		versions := r.computeOperatorStatusVersions(&operatorProgressingCondition, oldVersions, newVersions)
		versionsCmpOpts := []cmp.Option{
			cmpopts.EquateEmpty(),
			cmpopts.SortSlices(func(a, b configv1.OperandVersion) bool { return a.Name < b.Name }),
		}
		if !cmp.Equal(versions, expectedVersions, versionsCmpOpts...) {
			t.Errorf("%q: expected %v, got %v", tc.description, expectedVersions, versions)
		}
	}
}
