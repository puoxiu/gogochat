package kafka

import (
	"context"
	"time"

	"github.com/puoxiu/gogochat/pkg/zlog"
	"github.com/segmentio/kafka-go"
)

var ctx = context.Background()

type kafkaService struct {
	ChatWriter *kafka.Writer
	ChatReader *kafka.Reader
	KafkaConn  *kafka.Conn
	// 保存基础配置用于关闭连接
	addr string
}

var KafkaService = new(kafkaService)

// Init 初始化kafka，直接传入所需参数
// addr: kafka地址（如"localhost:9092"）
// topic: 聊天主题
// timeout: 超时时间（秒）
// groupID: 消费组ID
func (k *kafkaService) Init(addr, topic string, timeout time.Duration, groupID string) {
	k.addr = addr // 保存地址用于后续可能的连接操作

	// 初始化写入器
	k.ChatWriter = &kafka.Writer{
		Addr:                   kafka.TCP(addr),
		Topic:                  topic,
		Balancer:               &kafka.Hash{}, // 相同Key的消息发送到同一分区
		WriteTimeout:           timeout * time.Second,
		RequiredAcks:           kafka.RequireNone,
		AllowAutoTopicCreation: false, // 禁止自动创建topic，必须显式调用CreateTopic
	}

	// 初始化读取器
	k.ChatReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{addr},
		Topic:          topic,
		CommitInterval: timeout * time.Second,
		GroupID:        groupID,
		StartOffset:    kafka.LastOffset,
	})
}

// Close 关闭kafka连接
func (k *kafkaService) Close() {
	if err := k.ChatWriter.Close(); err != nil {
		zlog.Error(err.Error())
	}
	if err := k.ChatReader.Close(); err != nil {
		zlog.Error(err.Error())
	}
	if k.KafkaConn != nil {
		if err := k.KafkaConn.Close(); err != nil {
			zlog.Error(err.Error())
		}
	}
}

// CreateTopic 显式创建topic
// addr: kafka地址
// topic: 主题名称
// partitions: 分区数量
func (k *kafkaService) CreateTopic(addr, topic string, partitions int) error {
	// 连接Kafka节点
	var err error
	k.KafkaConn, err = kafka.Dial("tcp", addr)
	if err != nil {
		zlog.Error("连接Kafka失败: " + err.Error())
		return err
	}

	// 定义topic配置
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: 1,
		},
	}

	// 创建topic
	if err = k.KafkaConn.CreateTopics(topicConfigs...); err != nil {
		zlog.Error("创建Topic失败: " + err.Error())
		return err
	}
	return nil
}