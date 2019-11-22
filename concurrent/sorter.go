package concurrent

import "sort"

type Sorter struct {
	item []interface{}
	call func(o1, o2 interface{}) bool
}

func NewSorter(item []interface{}, call func(o1, o2 interface{}) bool) Sorter {
	return Sorter{item, call}
}

func (self *Sorter) Len() int {
	return len(self.item)
}

func (self Sorter) Less(i, j int) bool {
	return self.call(self.item[i], self.item[j])
}

func (self Sorter) Swap(i, j int) {
	self.item[i], self.item[j] = self.item[j], self.item[i]
}

func (self Sorter) Sort() []interface{} {
	sort.Sort(&self)
	return self.item
}
