package sarama

import "time"

type OffsetCommitResponse struct {
	Version        int16
	ThrottleTimeMs int32
	Errors         map[string]map[int32]KError
}

func (r *OffsetCommitResponse) setVersion(v int16) {
	r.Version = v
}

func (r *OffsetCommitResponse) AddError(topic string, partition int32, kerror KError) {
	if r.Errors == nil {
		r.Errors = make(map[string]map[int32]KError)
	}
	partitions := r.Errors[topic]
	if partitions == nil {
		partitions = make(map[int32]KError)
		r.Errors[topic] = partitions
	}
	partitions[partition] = kerror
}

func (r *OffsetCommitResponse) encode(pe packetEncoder) error {
	if r.Version >= 3 {
		pe.putInt32(r.ThrottleTimeMs)
	}
	if err := pe.putArrayLength(len(r.Errors)); err != nil {
		return err
	}
	for topic, partitions := range r.Errors {
		if err := pe.putString(topic); err != nil {
			return err
		}
		if err := pe.putArrayLength(len(partitions)); err != nil {
			return err
		}
		for partition, kerror := range partitions {
			pe.putInt32(partition)
			pe.putInt16(int16(kerror))
		}
	}
	return nil
}

func (r *OffsetCommitResponse) decode(pd packetDecoder, version int16) (err error) {
	r.Version = version

	if version >= 3 {
		r.ThrottleTimeMs, err = pd.getInt32()
		if err != nil {
			return err
		}
	}

	numTopics, err := pd.getArrayLength()
	if err != nil || numTopics == 0 {
		return err
	}

	r.Errors = make(map[string]map[int32]KError, numTopics)
	for i := 0; i < numTopics; i++ {
		name, err := pd.getString()
		if err != nil {
			return err
		}

		numErrors, err := pd.getArrayLength()
		if err != nil {
			return err
		}

		r.Errors[name] = make(map[int32]KError, numErrors)

		for j := 0; j < numErrors; j++ {
			id, err := pd.getInt32()
			if err != nil {
				return err
			}

			tmp, err := pd.getInt16()
			if err != nil {
				return err
			}
			r.Errors[name][id] = KError(tmp)
		}
	}

	return nil
}

func (r *OffsetCommitResponse) key() int16 {
	return apiKeyOffsetCommit
}

func (r *OffsetCommitResponse) version() int16 {
	return r.Version
}

func (r *OffsetCommitResponse) headerVersion() int16 {
	return 0
}

func (r *OffsetCommitResponse) isValidVersion() bool {
	return r.Version >= 0 && r.Version <= 7
}

func (r *OffsetCommitResponse) requiredVersion() KafkaVersion {
	switch r.Version {
	case 7:
		return V2_3_0_0
	case 5, 6:
		return V2_1_0_0
	case 4:
		return V2_0_0_0
	case 3:
		return V0_11_0_0
	case 2:
		return V0_9_0_0
	case 0, 1:
		return V0_8_2_0
	default:
		return V2_4_0_0
	}
}

func (r *OffsetCommitResponse) throttleTime() time.Duration {
	return time.Duration(r.ThrottleTimeMs) * time.Millisecond
}
