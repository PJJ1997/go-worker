package main

import "github.com/jrallison/go-workers"

/*
	Enqueue - 任务入队
		如果任务需要立即执行，则将任务信息保存到redis的queue-name队列中
		如果任务需要延迟执行，则将任务保存到公用的延迟zset队列中，并以待发送时间戳为score
	Process - 任务处理
		manger - 每个queue-name均需要创建一个manager，其负责任务的获取、调度、分发、收尾处理
			manager启动时候，需要检查是否有残留任务需要处理，也就是任务处理到一半，该process挂掉
			，导致任务未执行完毕，这些任务需要重新执行正常情况下，manager通过fetcher以brpoplpush
			形式，将待执行任务从queue-name转移到正在执行inprogress队列，同时通过golang自带chan将
			任务下发给worker
		worker - 从chan获取任务并执行，并通过middleware形式将处理状态返回，如果处理成功，则通过
			chan通知manager将任务从inprogress队列删除，否则，如果任务需要重试，则将重试信息放入retry队列，
			重试间隔时间成指数级递增
		schedule - 负责延迟、重试任务处理
			延迟任务 - 由于zset中score为任务执行时间戳，利用zrangebyscore，score为-inf -> now，
				索取可执行任务，将任务从zset中删除，并放入对应queue-name队列
			重试任务 - 处理过程与延迟任务相同
		fetcher - 从redis中相关队列获取任务，ack后删除任务等
*/

func main() {
	/*
		在config中设置redis相关信息 生产者：可以将任务放置于特定的queue_name，
		任务可以设置立即执行或者延迟执行 消费者：由管理器启动多个消费者，
		消费queue_name的任务并执行job函数
	*/
	workers.Configure(map[string]string{
		// location of redis instance
		"server": "localhost:6379",
		// instance of the database
		"database": "0",
		// number of connections to keep open with redis
		"pool": "30",
		// unique process id for this instance of workers (for proper recovery of inprogress jobs on crash)
		"process": "1",
	})

	// pull messages from "myqueue2" with concurrency of 20
	workers.Process("myqueue2", myJob, 20)
	// Add messages to a queue with retry
	workers.EnqueueWithOptions("myqueue2", "Add", []int{1, 2}, workers.EnqueueOptions{Retry: true})
	workers.Run()
}

func myJob(message *workers.Msg) {
	// do something with your message
	// message.Jid() 使用rand()方法生成的对于每个message的唯一标识，有可能会重复
	// message.Args() 可以转换成 map array json string int float64 等类型
	// message.Get("retry_count").Int() 获取重试次数，默认25次
	// 【重点】一旦 panic 会自动触发重试，一旦 return 会结束重试
}
