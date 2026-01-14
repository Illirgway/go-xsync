/**
 * This file is part of the go-xfmt package (https://github.com/Illirgway/go-xsync)
 *
 * Copyright (c) 2026 Illirgway
 *
 * This program is free software: you can redistribute it and/or modify it under the terms of the GNU
 * General Public License as published by the Free Software Foundation, either version 3 of the License,
 * or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
 * without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 * See the GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along with this program.
 * If not, see <https://www.gnu.org/licenses/>.
 *
 */

package xsync

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ABMutex locks either A-type or B-type threads in favor of opposite

type ABMutex struct {
	// NOTE x > 0 for A, x < 0 for B
	x int64 // atomic CaS
	// used to engage underlying semaphores / spinlocks
	a sync.RWMutex
	b sync.RWMutex
}

func (m *ABMutex) LockA() {

	for {
		x := atomic.LoadInt64(&m.x)

		// x > 0 for A (+1 for lock, -1 for unlock)
		if x >= 0 {
			if atomic.CompareAndSwapInt64(&m.x, x, x+1) {

				// успешно захватили лок под себя, должны теперь заблочить противостоящий тип задач, если первый захват с 0
				if x == 0 {
					m.b.Lock()
				}

				return
			}

			// провал CaS - тут же пытаемся сделать это снова (вдруг конкурирующий процесс нашего типа нас просто опередил?)
			continue
		}

		// ASSERT x < 0, вариант x >= 0 + !CaS отсеян выше
		// TODO check assertion for race

		// ALGO более совершенный вариант самого примитивного runtime.Gosched() - заставляем планировщик
		// снять с нас процесс, пока лок не будет освобожден конкурирующими процессами типа B, и тут же сами его
		// освобождаем, когда получаем обратно управление, чтобы пойти на новый круг попытки захвата лока
		m.a.RLock()
		m.a.RUnlock()
	}
}

func (m *ABMutex) UnlockA() {

	// TODO check m.x > 0

	// x > 0 for A (+1 for lock, -1 for unlock)
	x := atomic.AddInt64(&m.x, -1)

	if x < 0 {
		// TODO? пробуем оставаться стабильными, вернем обратно блокировку
		//  atomic.AddInt64(&m.x, +1)
		panic(fmt.Errorf("ABMutex try to unlock not blocked type A (maybe is confused with B): %d", x))
	}

	if x == 0 { // последняя блокировка этого типа снята, можно разлочить противостоящий тип
		m.b.Unlock()
	}
}

func (m *ABMutex) LockB() {

	for {
		x := atomic.LoadInt64(&m.x)

		// x < 0 for B (-1 for lock, +1 for unlock)
		if x <= 0 {
			if atomic.CompareAndSwapInt64(&m.x, x, x-1) {

				// успешно захватили лок под себя, должны теперь заблочить противостоящий тип задач, если первый захват с 0
				if x == 0 {
					m.a.Lock()
				}

				return
			}

			// провал CaS - тут же пытаемся сделать это снова (вдруг конкурирующий процесс нашего типа нас просто опередил?)
			continue
		}

		// ASSERT x > 0, вариант x <= 0 + !CaS отсеян выше
		// TODO check assertion for race

		// ALGO более совершенный вариант самого примитивного runtime.Gosched() - заставляем планировщик
		// снять с нас процесс, пока лок не будет освобожден конкурирующими процессами типа A, и тут же сами его
		// освобождаем, когда получаем обратно управление, чтобы пойти на новый круг попытки захвата лока
		m.b.RLock()
		m.b.RUnlock()
	}
}

func (m *ABMutex) UnlockB() {

	// TODO check m.x < 0

	// x < 0 for B (-1 for lock, +1 for unlock)
	x := atomic.AddInt64(&m.x, +1)

	if x > 0 {
		// TODO? пробуем оставаться стабильными, вернем обратно блокировку
		//  atomic.AddInt64(&m.x, -1)
		panic(fmt.Errorf("ABMutex try to unlock not blocked type B (maybe is confused with A): %d", x))
	}

	if x == 0 { // последняя блокировка этого типа снята, можно разлочить противостоящий тип
		m.a.Unlock()
	}
}
