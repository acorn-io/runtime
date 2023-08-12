package v1

import (
	"testing"

	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestGrantsAggregateGrants(t *testing.T) {
	missing, granted := Grants("ns", Permissions{
		ServiceName: "foo",
		Rules: []PolicyRule{
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs: []string{"a", "b"},
				},
			},
		},
	}, Permissions{
		ServiceName: "foo",
		Rules: []PolicyRule{
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs: []string{"b"},
				},
			},
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs: []string{"a"},
				},
			},
		},
	})
	assert.Equal(t, Permissions{ServiceName: "foo"}, missing)
	assert.True(t, granted)
}

func TestSimplify(t *testing.T) {
	autogold.Expect(Permissions{Rules: []PolicyRule{
		{PolicyRule: rbacv1.PolicyRule{
			Verbs: []string{"a", "b"},
		}},
	}}).Equal(t, Simplify(Permissions{
		Rules: []PolicyRule{
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs: []string{"a"},
				},
			},
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs: []string{"b"},
				},
			},
		},
	}))

	autogold.Expect(Permissions{Rules: []PolicyRule{
		{PolicyRule: rbacv1.PolicyRule{
			Verbs:     []string{"a", "b"},
			Resources: []string{"x", "y", "z"},
		}},
	}}).Equal(t, Simplify(Permissions{
		Rules: []PolicyRule{
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"a"},
					Resources: []string{"x", "y"},
				},
			},
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"b"},
					Resources: []string{"x", "y"},
				},
			},

			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"a", "b"},
					Resources: []string{"x", "y", "z"},
				},
			},
		},
	}))

	autogold.Expect(Permissions{Rules: []PolicyRule{
		{PolicyRule: rbacv1.PolicyRule{
			Verbs: []string{"a", "b"},
		}},
		{PolicyRule: rbacv1.PolicyRule{
			Verbs:     []string{"b"},
			APIGroups: []string{"group-a"},
		}},
	}}).Equal(t, Simplify(Permissions{
		Rules: []PolicyRule{
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs: []string{"a"},
				},
			},
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"b"},
					APIGroups: []string{"group-a"},
				},
			},
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs: []string{"b"},
				},
			},
		},
	}))

	autogold.Expect(Permissions{Rules: []PolicyRule{
		{PolicyRule: rbacv1.PolicyRule{
			Verbs:     []string{"a", "b"},
			APIGroups: []string{"group-a", "group-b"},
		}},
	}}).Equal(t, Simplify(Permissions{
		Rules: []PolicyRule{
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"a"},
					APIGroups: []string{"group-a"},
				},
			},
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"b"},
					APIGroups: []string{"group-b"},
				},
			},
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"a"},
					APIGroups: []string{"group-b"},
				},
			},
			{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"b"},
					APIGroups: []string{"group-a"},
				},
			},
		},
	}))
}
