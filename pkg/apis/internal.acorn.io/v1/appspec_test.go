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

	autogold.Expect([]Permissions{
		{
			ServiceName: "bar",
			Rules: []PolicyRule{{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"c"},
					APIGroups: []string{"group-b", "group-c"},
				}},
			},
		},
		{
			ServiceName: "foo",
			Rules: []PolicyRule{{
				PolicyRule: rbacv1.PolicyRule{
					Verbs:     []string{"a", "b"},
					APIGroups: []string{"group-a", "group-b"},
				}},
			},
		},
	},
	).Equal(t, SimplifySet([]Permissions{
		{
			ServiceName: "foo",
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
			},
		},
		{
			ServiceName: "foo",
			Rules: []PolicyRule{
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
		},
		{
			ServiceName: "bar",
			Rules: []PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:     []string{"c"},
						APIGroups: []string{"group-c"},
					},
				},
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:     []string{"c"},
						APIGroups: []string{"group-b"},
					},
				},
			},
		},
		{
			ServiceName: "bar",
			Rules: []PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:     []string{"c"},
						APIGroups: []string{"group-c"},
					},
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

func TestGrantsAllClusterGrantsProject(t *testing.T) {
	requested := []Permissions{
		{
			ServiceName: "foo",
			Rules: []PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:     []string{"get"},
						APIGroups: []string{"group-a"},
					},
					Scopes: []string{"project"},
				},
			},
		},
	}

	granted := []Permissions{
		{
			ServiceName: "foo",
			Rules: []PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:     []string{"get"},
						APIGroups: []string{"group-a"},
					},
					Scopes: []string{"cluster"},
				},
			},
		},
	}
	gotMissing, _ := GrantsAll("acorn", requested, granted)
	assert.Equal(t, []Permissions(nil), gotMissing, "cluster permissions should grant project permissions")
}

func TestGrantsAllClusterGrantsNamespace(t *testing.T) {
	requested := []Permissions{
		{
			ServiceName: "foo",
			Rules: []PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:     []string{"get"},
						APIGroups: []string{"group-a"},
					},
					Scopes: []string{"namespace:bar"},
				},
			},
		},
	}

	granted := []Permissions{
		{
			ServiceName: "foo",
			Rules: []PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:     []string{"get"},
						APIGroups: []string{"group-a"},
					},
					Scopes: []string{"cluster"},
				},
			},
		},
	}
	gotMissing, _ := GrantsAll("acorn", requested, granted)
	assert.Equal(t, []Permissions(nil), gotMissing, "cluster permissions should grant namespace permissions")
}

func TestGrantsAllProjectNotGrantCluster(t *testing.T) {
	requested := []Permissions{
		{
			ServiceName: "foo",
			Rules: []PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:     []string{"get"},
						APIGroups: []string{"group-a"},
					},
					Scopes: []string{"cluster"},
				},
			},
		},
	}

	granted := []Permissions{
		{
			ServiceName: "foo",
			Rules: []PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:     []string{"get"},
						APIGroups: []string{"group-a"},
					},
					Scopes: []string{"project"},
				},
			},
		},
	}
	gotMissing, _ := GrantsAll("acorn", requested, granted)
	assert.Equal(t, requested, gotMissing, "project permissions should not grant cluster permissions")
}
