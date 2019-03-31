package selector

import (
	"context"
	"errors"
	"github.com/megaredfan/rpc-demo/protocol"
	"github.com/megaredfan/rpc-demo/registry"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var ErrEmptyProviderList = errors.New("provider list is empty")

type Filter func(provider registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) bool

type SelectOption struct {
	Filters []Filter
}

func DegradeProviderFilter() Filter {
	return func(provider registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) bool {
		_, degrade := provider.Meta[protocol.ProviderDegradeKey]
		return !degrade
	}
}

func TaggedProviderFilter(tags map[string]string) Filter {
	return func(provider registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) bool {
		if tags == nil {
			return true
		}
		if provider.Meta == nil {
			return false
		}
		providerTags, ok := provider.Meta["tags"].(map[string]string)
		if !ok || len(providerTags) <= 0 {
			return false
		}
		for k, v := range tags {
			if tag, ok := providerTags[k]; ok {
				if tag != v {
					return false
				}
			} else {
				return false
			}
		}
		return true
	}
}

type Selector interface {
	Next(providers []registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}, opt SelectOption) (registry.Provider, error)
}

type RandomSelector struct {
}

var RandomSelectorInstance = RandomSelector{}

func (RandomSelector) Next(providers []registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}, opt SelectOption) (p registry.Provider, err error) {
	filters := combineFilter(opt.Filters)
	list := make([]registry.Provider, 0)
	for _, p := range providers {
		if filters(p, ctx, ServiceMethod, arg) {
			list = append(list, p)
		}
	}

	if len(list) == 0 {
		err = ErrEmptyProviderList
		return
	}
	i := rand.Intn(len(list))
	p = list[i]
	return
}

func combineFilter(filters []Filter) Filter {
	return func(provider registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) bool {
		for _, f := range filters {
			if !f(provider, ctx, ServiceMethod, arg) {
				return false
			}
		}
		return true
	}
}

func NewRandomSelector() Selector {
	return RandomSelectorInstance
}
