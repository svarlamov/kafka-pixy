package pixy

import (
	"strconv"

	. "github.com/mailgun/kafka-pixy/Godeps/_workspace/src/gopkg.in/check.v1"
	"github.com/mailgun/kafka-pixy/config"
	"github.com/mailgun/kafka-pixy/logging"
)

type AdminSuite struct {
	config *config.T
}

var _ = Suite(&AdminSuite{})

func (s *AdminSuite) SetUpSuite(c *C) {
	logging.InitTest()
	s.config = config.Default()
	s.config.ClientID = "producer"
	s.config.Kafka.SeedPeers = testKafkaPeers
	s.config.ZooKeeper.SeedPeers = testZookeeperPeers
}

// The end offset of partition ranges is properly reflects the number of
// messages produced since the previous check.
func (s *AdminSuite) TestGetOffsetsAfterProduce(c *C) {
	// Given
	keyToCount := make(map[string]int, 64)
	for i := 0; i < 64; i++ {
		keyToCount[strconv.Itoa(i)] = i
	}
	GenMessages(c, "get_offsets", "test.64", keyToCount)

	a, err := SpawnAdmin(s.config)
	c.Assert(err, IsNil)
	offsetsBefore, err := a.GetGroupOffsets("foo", "test.64")
	c.Assert(err, IsNil)
	GenMessages(c, "get_offsets", "test.64", keyToCount)

	// When
	offsetsAfter, err := a.GetGroupOffsets("foo", "test.64")
	c.Assert(err, IsNil)

	// Then
	rangeEndDiffs := []int{
		0, 75, 3, 0, 57, 0, 0, 8, 58, 6, 30, 59, 0, 75, 63, 0, 48, 37, 49, 0,
		75, 2, 0, 51, 0, 0, 0, 79, 1, 33, 61, 39, 75, 62, 0, 4, 36, 0, 0, 79,
		61, 28, 53, 35, 18, 0, 79, 0, 32, 63, 38, 0, 9, 59, 7, 31, 14, 0, 79, 60,
		29, 55, 34, 67,
	}
	for i := 0; i < 64; i++ {
		actualDiff := int(offsetsAfter[i].End - offsetsBefore[i].End)
		c.Assert(actualDiff, Equals, rangeEndDiffs[i])
	}
	a.Stop()
}

// It is possible to set offsets for only a subset of group/topic partitions.
func (s *AdminSuite) TestSetOffsetsPartialUpdate(c *C) {
	// Given
	a, err := SpawnAdmin(s.config)
	c.Assert(err, IsNil)
	a.SetGroupOffsets("foo", "test.4", []PartitionOffset{
		{Partition: 0, Offset: 1001, Metadata: "A1"},
		{Partition: 1, Offset: 1002, Metadata: "A2"},
		{Partition: 2, Offset: 1003, Metadata: "A3"},
		{Partition: 3, Offset: 1004, Metadata: "A4"},
	})

	// When
	a.SetGroupOffsets("foo", "test.4", []PartitionOffset{
		{Partition: 0, Offset: 2001, Metadata: "B1"},
		{Partition: 3, Offset: 2004, Metadata: "B4"},
	})

	// Then
	offsets, err := a.GetGroupOffsets("foo", "test.4")
	c.Assert(offsets[0].Offset, Equals, int64(2001))
	c.Assert(offsets[1].Offset, Equals, int64(1002))
	c.Assert(offsets[2].Offset, Equals, int64(1003))
	c.Assert(offsets[3].Offset, Equals, int64(2004))
	c.Assert(offsets[0].Metadata, Equals, "B1")
	c.Assert(offsets[1].Metadata, Equals, "A2")
	c.Assert(offsets[2].Metadata, Equals, "A3")
	c.Assert(offsets[3].Metadata, Equals, "B4")

	a.Stop()
}
