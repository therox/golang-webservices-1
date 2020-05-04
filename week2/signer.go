package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// сюда писать код

// SingleHash считает значение crc32(data)+"~"+crc32(md5(data)) ( конкатенация двух строк через ~),
// где data - то что пришло на вход (по сути - числа из первой функции)
func SingleHash(in, out chan interface{}) {
	// Получаем по-очерёдно все значения из входящего канала
	// Для работы с DataSignerMd5 нужно создать очередь

	dsmd5in := make(chan string)
	dsmd5out := make(chan string)
	//defer close(dsmd5in)
	//defer close(dsmd5out)

	go func(in chan string, out chan string) {
		for {
			select {
			case data := <-in:
				out <- DataSignerMd5(data)
			default:
			}
		}
	}(dsmd5in, dsmd5out)

	gWg := &sync.WaitGroup{}
	for inData := range in {
		time.Sleep(1 * time.Millisecond)
		curData := fmt.Sprintf("%v", inData)
		gWg.Add(1)
		go func(data string, md5In, md5Out chan string, gwg *sync.WaitGroup) {
			var left, right string
			var wg = &sync.WaitGroup{}
			wg.Add(2)
			go func() {
				left = DataSignerCrc32(curData)
				wg.Done()
			}()
			go func() {
				md5In <- curData
				right = DataSignerCrc32(<-md5Out)
				wg.Done()
			}()
			wg.Wait()

			out <- left + "~" + right
			gWg.Done()
		}(curData, dsmd5in, dsmd5out, gWg)
	}
	gWg.Wait()
}

// MultiHash считает значение crc32(th+data)) (конкатенация цифры, приведённой к строке и строки),
// где th=0..5 ( т.е. 6 хешей на каждое входящее значение ), потом берёт конкатенацию результатов в порядке
// расчета (0..5), где data - то что пришло на вход (и ушло на выход из SingleHash)
func MultiHash(in, out chan interface{}) {
	var gResults = struct {
		mx   *sync.Mutex
		data []string
	}{
		new(sync.Mutex),
		make([]string, 0),
	}
	gWg := &sync.WaitGroup{}
	j := 0
	for inData := range in {
		//mx := &sync.Mutex{}
		gResults.mx.Lock()
		gResults.data = append(gResults.data, "")
		gResults.mx.Unlock()
		results := make([]string, 6)
		curData := fmt.Sprintf("%v", inData)

		gWg.Add(1)
		go func(j int) {
			defer gWg.Done()
			wg := &sync.WaitGroup{}
			wg.Add(6)
			for i := 0; i < 6; i++ {
				go func(index int) {
					results[index] = DataSignerCrc32(fmt.Sprintf("%d%s", index, curData))
					wg.Done()
				}(i)
			}
			wg.Wait()
			gResults.mx.Lock()
			gResults.data[j] = strings.Join(results, "")
			gResults.mx.Unlock()
		}(j)
		j++
	}
	gWg.Wait()
	for i := range gResults.data {
		out <- gResults.data[i]
	}

}

// CombineResults получает все результаты, сортирует (https://golang.org/pkg/sort/),
// объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
func CombineResults(in, out chan interface{}) {
	var results = make([]string, 0)
	for inData := range in {
		results = append(results, fmt.Sprintf("%v", inData))
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})
	out <- strings.Join(results, "_")

}

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})
	defer close(in)
	for i := range jobs {
		out := make(chan interface{})
		wg.Add(1)
		go func(jobFunc job, in, out chan interface{}) {
			jobFunc(in, out)
			close(out)
			wg.Done()
		}(jobs[i], in, out)
		in = out
	}
	wg.Wait()
}
