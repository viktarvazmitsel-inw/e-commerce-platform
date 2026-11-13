package broker

import "fmt"

type BrokerServiceList struct {
	List []string
}

func NewBrokerServiceList(services []string) *BrokerServiceList {
	return &BrokerServiceList{
		List: services,
	}
}

func (l *BrokerServiceList) ValidateService(service string) error {
	for _, s := range l.List {
		if s == service {
			return nil
		}
	}
	return fmt.Errorf("service %s not found", service)
}
