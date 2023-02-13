package serializer

import "k8s.io/apimachinery/pkg/runtime"

type NoProtobufSerializer struct {
	r runtime.NegotiatedSerializer
}

func NewNoProtobufSerializer(r runtime.NegotiatedSerializer) runtime.NegotiatedSerializer {
	return &NoProtobufSerializer{
		r: r,
	}
}

func (n *NoProtobufSerializer) SupportedMediaTypes() []runtime.SerializerInfo {
	si := n.r.SupportedMediaTypes()
	result := make([]runtime.SerializerInfo, 0, len(si))
	for _, s := range si {
		if s.MediaType == runtime.ContentTypeProtobuf {
			continue
		}
		result = append(result, s)
	}
	return result
}

func (n *NoProtobufSerializer) EncoderForVersion(serializer runtime.Encoder, gv runtime.GroupVersioner) runtime.Encoder {
	return n.r.EncoderForVersion(serializer, gv)
}

func (n *NoProtobufSerializer) DecoderToVersion(serializer runtime.Decoder, gv runtime.GroupVersioner) runtime.Decoder {
	return n.r.DecoderToVersion(serializer, gv)
}
