package curly

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

var formatters sync.Map

// Formatter formats an object into a string
type Formatter interface {
	Format(v interface{}) (string, error)
}

// Format an object into a string based on the format.
func Format(format string, source interface{}) (string, error) {
	f, err := NewFormatter(format, source)
	if err != nil {
		return "", err
	}
	return f.Format(source)
}

// NewFormatter creaets a formatter object that can be reused to format the
// same objects.
func NewFormatter(format string, source interface{}) (Formatter, error) {
	type key struct {
		format string
		source reflect.Type
	}
	t := reflect.TypeOf(source)
	var k interface{} = key{format, t}
	v, loaded := formatters.Load(k)
	if loaded {
		return v.(Formatter), nil
	}
	f, err := newFormatter(format, t)
	if err != nil {
		return nil, err
	}
	v = f
	v, loaded = formatters.LoadOrStore(k, v)
	if loaded {
		return v.(Formatter), nil
	}
	return f, nil
}

type formatter struct {
	pairs []formatPair
}

// Format an object.
func (f *formatter) Format(v interface{}) (string, error) {
	fragments := make([]string, 0, len(f.pairs)*2)
	for _, p := range f.pairs {
		s, err := p.Format(v)
		if err != nil {
			return "", err
		}
		fragments = append(fragments, p.prefix, s)
	}
	return strings.Join(fragments, ""), nil
}

type nopFormatter struct{ string }

func (f nopFormatter) Format(v interface{}) (string, error) {
	return f.string, nil
}

var curlyRegex = regexp.MustCompile(`\{[^\}]*\}`)

func newFormatter(format string, source reflect.Type) (Formatter, error) {
	beginEnds := curlyRegex.FindAllStringIndex(format, -1)
	if len(beginEnds) == 0 {
		return nopFormatter{format}, nil
	}
	f := &formatter{
		pairs: make([]formatPair, 0, len(beginEnds)+1),
	}
	index := 0
	for _, be := range beginEnds {
		begin, end := be[0], be[1]
		selectors, err := getSelectors(
			source,
			strings.Split(format[begin+1:end-1], "."))
		if err != nil {
			return nil, err
		}
		f.pairs = append(f.pairs, formatPair{
			prefix:    format[index:begin],
			selectors: selectors,
		})
		index = end
	}
	f.pairs = append(f.pairs, formatPair{
		prefix: format[index:],
	})
	return f, nil
}

type selector func(interface{}) (interface{}, error)

func getSelectors(source reflect.Type, path []string) ([]selector, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("no error path")
	}
	selectors := make([]selector, len(path))
	for i, p := range path {
		if source.Kind() == reflect.Struct {
			f, ok := source.FieldByNameFunc(func(name string) bool {
				return strings.EqualFold(name, p)
			})
			if ok {
				source = f.Type
				selectors[i] = func(v interface{}) (interface{}, error) {
					rv := reflect.ValueOf(v)
					return rv.FieldByIndex(f.Index).Interface(), nil
				}
				continue
			}
		}
		methods := source.NumMethod()
		for i := 0; i < methods; i++ {
			m := source.Method(i)
			if !strings.EqualFold(m.Name, p) {
				continue
			}
			no := m.Type.NumOut()
			if no == 0 || no > 2 {
				return nil, fmt.Errorf(
					"method %q on type %v has %d return "+
						"values",
					m.Name, source.Name(), no)
			}
			source = m.Type.Out(0)
			selectors[i] = func(v interface{}) (interface{}, error) {
				var temp [1]reflect.Value
				temp[0] = reflect.ValueOf(v)
				rvs := m.Func.Call(temp[:])
				if no == 2 {
					err, ok := rvs[1].Interface().(error)
					if ok {
						return nil, err
					}
				}
				return rvs[0].Interface(), nil
			}
		}
	}
	return selectors, nil
}

type formatPair struct {
	prefix    string
	selectors []selector
}

func (p formatPair) Format(v interface{}) (string, error) {
	if len(p.selectors) == 0 {
		return "", nil
	}
	var err error
	for _, selector := range p.selectors {
		if v, err = selector(v); err != nil {
			return "", err
		}
	}
	s := fmt.Sprint(v)
	return s, nil
}

type action struct {
	formatter string
}
