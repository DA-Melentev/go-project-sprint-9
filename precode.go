package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	n := int64(1)
	for {
		select {
		case <-ctx.Done(): // Завершаем при отмене контекста
			close(ch) // Закрываем канал перед выходом
			return
		default:
			ch <- n // Отправляем число в канал
			fn(n)   // Подсчитываем сумму и количество
			n++     // Увеличиваем на 1
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	for num := range in { // Читаем из канала, пока он не закроется
		out <- num                       // Пишем в выходной канал
		time.Sleep(1 * time.Millisecond) // Имитация обработки
	}
	close(out) // Закрываем выходной канал после обработки
}

func main() {
	chIn := make(chan int64)

	// 3. Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Счетчики для проверки
	var inputSum int64
	var inputCount int64

	// Запускаем генератор чисел
	go Generator(ctx, chIn, func(i int64) {
		atomic.AddInt64(&inputSum, i)
		atomic.AddInt64(&inputCount, 1)
	})

	const NumOut = 5
	outs := make([]chan int64, NumOut)

	// Создаем и запускаем Worker горутины
	for i := 0; i < NumOut; i++ {
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	// Сбор результатов из каналов outs в результирующий канал
	chOut := make(chan int64, NumOut)
	var wg sync.WaitGroup

	for i, out := range outs {
		wg.Add(1)
		go func(in <-chan int64, i int) {
			defer wg.Done()
			for num := range in {
				chOut <- num
			}
		}(out, i)
	}

	// Закрываем результирующий канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(chOut)
	}()

	// Подсчет итоговых значений
	var count int64
	var sum int64

	for num := range chOut {
		atomic.AddInt64(&count, 1)
		atomic.AddInt64(&sum, num)
	}

	// Вывод результатов
	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)

	// Проверка корректности
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
}
