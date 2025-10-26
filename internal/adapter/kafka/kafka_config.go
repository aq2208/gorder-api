package kafka

import (
	"time"

	"github.com/IBM/sarama"
)

func NewGroup(brokers []string, groupID string) (sarama.ConsumerGroup, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_6_0_0
	cfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.Net.DialTimeout = 5 * time.Second
	return sarama.NewConsumerGroup(brokers, groupID, cfg)
}
