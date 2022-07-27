package store

type Store interface {
	StoreWorkload
	StoreContainer
	StoreNode
}

type StoreWorkload interface {
	StoreWorkloadSpec()
	StoreWorkloadTimeSeries()
}

type StoreContainer interface {
	StoreContainerSpec()
	StoreContainerTimeSeries()
}

type StoreNode interface {
	StoreNodeSpec()
	StoreNodeTimeSeries()
}
