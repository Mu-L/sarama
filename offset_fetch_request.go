package sarama

type OffsetFetchRequest struct {
	Version       int16
	ConsumerGroup string
	RequireStable bool // requires v7+
	partitions    map[string][]int32
}

func (r *OffsetFetchRequest) setVersion(v int16) {
	r.Version = v
}

func NewOffsetFetchRequest(
	version KafkaVersion,
	group string,
	partitions map[string][]int32,
) *OffsetFetchRequest {
	request := &OffsetFetchRequest{
		ConsumerGroup: group,
		partitions:    partitions,
	}
	if version.IsAtLeast(V2_5_0_0) {
		// Version 7 is adding the require stable flag.
		request.Version = 7
	} else if version.IsAtLeast(V2_4_0_0) {
		// Version 6 is the first flexible version.
		request.Version = 6
	} else if version.IsAtLeast(V2_1_0_0) {
		// Version 3, 4, and 5 are the same as version 2.
		request.Version = 5
	} else if version.IsAtLeast(V2_0_0_0) {
		request.Version = 4
	} else if version.IsAtLeast(V0_11_0_0) {
		request.Version = 3
	} else if version.IsAtLeast(V0_10_2_0) {
		// Starting in version 2, the request can contain a null topics array to indicate that offsets
		// for all topics should be fetched. It also returns a top level error code
		// for group or coordinator level errors.
		request.Version = 2
	} else if version.IsAtLeast(V0_8_2_0) {
		// In version 0, the request read offsets from ZK.
		//
		// Starting in version 1, the broker supports fetching offsets from the internal __consumer_offsets topic.
		request.Version = 1
	}

	return request
}

func (r *OffsetFetchRequest) encode(pe packetEncoder) (err error) {
	if r.Version < 0 || r.Version > 7 {
		return PacketEncodingError{"invalid or unsupported OffsetFetchRequest version field"}
	}

	isFlexible := r.Version >= 6

	if isFlexible {
		err = pe.putCompactString(r.ConsumerGroup)
	} else {
		err = pe.putString(r.ConsumerGroup)
	}
	if err != nil {
		return err
	}

	if isFlexible {
		if r.partitions == nil {
			pe.putUVarint(0)
		} else {
			pe.putCompactArrayLength(len(r.partitions))
		}
	} else {
		if r.partitions == nil && r.Version >= 2 {
			pe.putInt32(-1)
		} else {
			if err = pe.putArrayLength(len(r.partitions)); err != nil {
				return err
			}
		}
	}

	for topic, partitions := range r.partitions {
		if isFlexible {
			err = pe.putCompactString(topic)
		} else {
			err = pe.putString(topic)
		}
		if err != nil {
			return err
		}

		//

		if isFlexible {
			err = pe.putCompactInt32Array(partitions)
		} else {
			err = pe.putInt32Array(partitions)
		}
		if err != nil {
			return err
		}

		if isFlexible {
			pe.putEmptyTaggedFieldArray()
		}
	}

	if r.RequireStable && r.Version < 7 {
		return PacketEncodingError{"requireStable is not supported. use version 7 or later"}
	}

	if r.Version >= 7 {
		pe.putBool(r.RequireStable)
	}

	if isFlexible {
		pe.putEmptyTaggedFieldArray()
	}

	return nil
}

func (r *OffsetFetchRequest) decode(pd packetDecoder, version int16) (err error) {
	r.Version = version
	isFlexible := r.Version >= 6
	if isFlexible {
		r.ConsumerGroup, err = pd.getCompactString()
	} else {
		r.ConsumerGroup, err = pd.getString()
	}
	if err != nil {
		return err
	}

	var partitionCount int

	if isFlexible {
		partitionCount, err = pd.getCompactArrayLength()
	} else {
		partitionCount, err = pd.getArrayLength()
	}
	if err != nil {
		return err
	}

	if (partitionCount == 0 && version < 2) || partitionCount < 0 {
		return nil
	}

	r.partitions = make(map[string][]int32, partitionCount)
	for i := 0; i < partitionCount; i++ {
		var topic string
		if isFlexible {
			topic, err = pd.getCompactString()
		} else {
			topic, err = pd.getString()
		}
		if err != nil {
			return err
		}

		var partitions []int32
		if isFlexible {
			partitions, err = pd.getCompactInt32Array()
		} else {
			partitions, err = pd.getInt32Array()
		}
		if err != nil {
			return err
		}
		if isFlexible {
			_, err = pd.getEmptyTaggedFieldArray()
			if err != nil {
				return err
			}
		}

		r.partitions[topic] = partitions
	}

	if r.Version >= 7 {
		r.RequireStable, err = pd.getBool()
		if err != nil {
			return err
		}
	}

	if isFlexible {
		_, err = pd.getEmptyTaggedFieldArray()
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *OffsetFetchRequest) key() int16 {
	return apiKeyOffsetFetch
}

func (r *OffsetFetchRequest) version() int16 {
	return r.Version
}

func (r *OffsetFetchRequest) headerVersion() int16 {
	if r.Version >= 6 {
		return 2
	}

	return 1
}

func (r *OffsetFetchRequest) isValidVersion() bool {
	return r.Version >= 0 && r.Version <= 7
}

func (r *OffsetFetchRequest) requiredVersion() KafkaVersion {
	switch r.Version {
	case 7:
		return V2_5_0_0
	case 6:
		return V2_4_0_0
	case 5:
		return V2_1_0_0
	case 4:
		return V2_0_0_0
	case 3:
		return V0_11_0_0
	case 2:
		return V0_10_2_0
	case 1:
		return V0_8_2_0
	case 0:
		return V0_8_2_0
	default:
		return V2_5_0_0
	}
}

func (r *OffsetFetchRequest) ZeroPartitions() {
	if r.partitions == nil && r.Version >= 2 {
		r.partitions = make(map[string][]int32)
	}
}

func (r *OffsetFetchRequest) AddPartition(topic string, partitionID int32) {
	if r.partitions == nil {
		r.partitions = make(map[string][]int32)
	}

	r.partitions[topic] = append(r.partitions[topic], partitionID)
}
