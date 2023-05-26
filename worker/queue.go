/*
Copyright 2014 Workiva, LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package queue includes a regular queue and a priority queue.
These queues rely on waitgroups to pause listening threads
on empty queues until a message is received.  If any thread
calls Dispose on the queue, any listeners are immediately returned
with an error.  Any subsequent put to the queue will return an error
as opposed to panicking as with channels.  Queues will grow with unbounded
behavior as opposed to channels which can be buffered but will pause
while a thread attempts to put to a full channel.

Recently added is a lockless ring buffer using the same basic C design as
found here:

http://www.1024cores.net/home/lock-free-algorithms/queues/bounded-mpmc-queue

Modified for use with Go with the addition of some dispose semantics providing
the capability to release blocked threads.  This works for both puts
and gets, either will return an error if they are blocked and the buffer
is disposed.  This could serve as a signal to kill a goroutine.  All threadsafety
is acheived using CAS operations, making this buffer pretty quick.

Benchmarks:
BenchmarkPriorityQueue-8	 		2000000	       782 ns/op
BenchmarkQueue-8	 		 		2000000	       671 ns/op
BenchmarkChannel-8	 		 		1000000	      2083 ns/op
BenchmarkQueuePut-8	   		   		20000	     84299 ns/op
BenchmarkQueueGet-8	   		   		20000	     80753 ns/op
BenchmarkExecuteInParallel-8	    20000	     68891 ns/op
BenchmarkRBLifeCycle-8				10000000	       177 ns/op
BenchmarkRBPut-8					30000000	        58.1 ns/op
BenchmarkRBGet-8					50000000	        26.8 ns/op

TODO: We really need a Fibonacci heap for the priority queue.
TODO: Unify the types of queue to the same interface.
*/
package worker

type Queue struct {
	data []interface{}
}

func (q *Queue) Put(k interface{}) {
	q.data = append(q.data, k)
}

func (q *Queue) Get() interface{} {
	if len(q.data) == 0 {
		return nil
	}
	v := q.data[0]
	q.data = q.data[1:]
	return v
}

func (q *Queue) Len() int {
	return len(q.data)
}

func New() *Queue {
	return &Queue{
		data: make([]interface{}, 0),
	}
}
