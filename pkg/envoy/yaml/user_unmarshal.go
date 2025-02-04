package yaml

import (
	"github.com/cortezaproject/corteza-server/pkg/envoy"
	"github.com/cortezaproject/corteza-server/pkg/envoy/resource"
	"github.com/cortezaproject/corteza-server/pkg/y7s"
	"github.com/cortezaproject/corteza-server/system/types"
	"gopkg.in/yaml.v3"
)

func (wset *userSet) UnmarshalYAML(n *yaml.Node) error {
	return y7s.Each(n, func(k, v *yaml.Node) (err error) {
		var (
			wrap = &user{}
		)

		if v == nil {
			return y7s.NodeErr(n, "malformed user definition")
		}

		wrap.res = &types.User{
			EmailConfirmed: true,
		}

		switch v.Kind {
		case yaml.ScalarNode:
			if err = y7s.DecodeScalar(v, "user email", &wrap.res.Email); err != nil {
				return
			}

		case yaml.MappingNode:
			if err = v.Decode(&wrap); err != nil {
				return
			}

		default:
			return y7s.NodeErr(n, "expecting scalar or map with user definitions")

		}

		if err = decodeRef(k, "user", &wrap.res.Handle); err != nil {
			return err
		}

		*wset = append(*wset, wrap)
		return
	})
}

func (wrap *user) UnmarshalYAML(n *yaml.Node) (err error) {
	if !y7s.IsKind(n, yaml.MappingNode) {
		return y7s.NodeErr(n, "user definition must be a map")
	}

	if wrap.res == nil {
		wrap.res = &types.User{}
	}

	if err = n.Decode(&wrap.res); err != nil {
		return
	}

	if wrap.rbac, err = decodeRbac(n); err != nil {
		return
	}

	if wrap.envoyConfig, err = decodeEnvoyConfig(n); err != nil {
		return
	}

	if wrap.roles, err = decodeUserRoles(n); err != nil {
		return
	}

	if wrap.ts, err = decodeTimestamps(n); err != nil {
		return
	}

	return nil
}

func decodeUserRoles(n *yaml.Node) (roles []string, err error) {
	var ecNode *yaml.Node
	for i, k := range n.Content {
		if k.Value == "roles" {
			ecNode = n.Content[i+1]
			break
		}
	}

	if ecNode == nil {
		return
	}

	return roles, y7s.EachSeq(ecNode, func(v *yaml.Node) (err error) {
		roles = append(roles, v.Value)
		return nil
	})
}

func (wset userSet) MarshalEnvoy() ([]resource.Interface, error) {
	nn := make([]resource.Interface, 0, len(wset))

	for _, res := range wset {
		if tmp, err := res.MarshalEnvoy(); err != nil {
			return nil, err
		} else {
			nn = append(nn, tmp...)
		}

	}

	return nn, nil
}

func (wrap user) MarshalEnvoy() ([]resource.Interface, error) {
	rs := resource.NewUser(wrap.res, wrap.roles...)
	rs.SetTimestamps(wrap.ts)
	rs.SetConfig(wrap.envoyConfig)

	return envoy.CollectNodes(
		rs,
		wrap.rbac.bindResource(rs),
	)
}
