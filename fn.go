package main

import (
	"bytes"
	"context"
	"dario.cat/mergo"
	"encoding/base64"
	"fmt"
	"github.com/crossplane-contrib/function-ytt-templating/input/v1beta1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/json"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const (
	annotationKeyCompositionResourceName = "ytt.fn.crossplane.io/composition-resource-name"
	annotationKeyReady                   = "ytt.fn.crossplane.io/ready"

	metaApiVersion = "meta.ytt.fn.crossplane.io/v1alpha1"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1beta1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1beta1.RunFunctionRequest) (*fnv1beta1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	in := &v1beta1.YTT{}
	if err := request.GetInput(req, in); err != nil {
		f.log.Info("cannot get Function input from %T", req)
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	tg, err := NewTemplateSourceGetter(in)
	if err != nil {
		f.log.Info("invalid function input: %T", err.Error())
		response.Fatal(rsp, errors.Wrap(err, "invalid function input"))
		return rsp, nil
	}

	reqMap, err := convertToMap(req)
	if err != nil {
		f.log.Info("cannot convert request to map")
		response.Fatal(rsp, errors.Wrap(err, "cannot convert request to map"))
		return rsp, nil
	}

	f.log.Debug("constructed request map", "request", reqMap)
	marshalJSON, err := req.GetObserved().GetComposite().GetResource().MarshalJSON()
	if err != nil {
		f.log.Info("Cannot Unmarshal JSON: %T", err.Error())
		return nil, err
	}
	toYAML, err := yaml.JSONToYAML([]byte(marshalJSON))
	fmt.Println(string(toYAML))
	oxr, err := request.GetObservedCompositeResource(req)

	if err != nil {
		f.log.Info("cannot get observed XR from %T", req)
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed XR from %T", req))
		return rsp, nil
	}

	specVal, err := oxr.Resource.GetValue("spec")
	if err != nil {
		f.log.Info("cannot get spec: %T", err.Error())
		response.Fatal(rsp, errors.Wrapf(err, "cannot get spec: %T", err.Error()))
		return nil, err
	}
	specInYAML, err := yaml.Marshal(specVal)
	if err != nil {
		f.log.Info("cannot marshal to YAML: %T", err.Error())
		response.Fatal(rsp, errors.Wrapf(err, "cannot marshal to YAML: %T", err.Error()))
		return nil, err
	}
	header := []byte(`#@data/values
---
`)
	valuesFile := append(header, specInYAML...)
	fmt.Println(string(valuesFile))

	resp, err := ytt([]string{tg.GetTemplates(), string(valuesFile)})
	if err != nil {
		f.log.Info("error while executing ytt: %T", err.Error())
		response.Fatal(rsp, errors.Wrapf(err, "error while executing ytt: %T", err.Error()))
		return nil, err
	}
	f.log.Info("rendered manifests", "manifests", resp)

	// Parse the rendered manifests.
	var objs []*unstructured.Unstructured
	decoder := yaml2.NewYAMLOrJSONDecoder(bytes.NewBufferString(resp), 1024)
	for {
		u := &unstructured.Unstructured{}
		if err := decoder.Decode(&u); err != nil {
			if err == io.EOF {
				break
			}
			response.Fatal(rsp, errors.Wrap(err, "cannot decode manifest"))
			return rsp, nil
		}
		if u != nil {
			objs = append(objs, u)
		}
	}

	// Get the desired composite resource from the request.
	desiredComposite, err := request.GetDesiredCompositeResource(req)
	if err != nil {
		f.log.Info("cannot get desired composite resource")
		response.Fatal(rsp, errors.Wrap(err, "cannot get desired composite resource"))
		return rsp, nil
	}

	// Get the observed composite resource from the request.
	observedComposite, err := request.GetObservedCompositeResource(req)
	if err != nil {
		f.log.Info("cannot get observed composite resource")
		response.Fatal(rsp, errors.Wrap(err, "cannot get observed composite resource"))
		return rsp, nil
	}

	//  Get the desired composed resources from the request.
	desiredComposed, err := request.GetDesiredComposedResources(req)
	if err != nil {
		f.log.Info("cannot get desired composed resources")
		response.Fatal(rsp, errors.Wrap(err, "cannot get desired composed resources"))
		return rsp, nil
	}

	// Convert the rendered manifests to a list of desired composed resources.
	for _, obj := range objs {
		cd := resource.NewDesiredComposed()
		cd.Resource.Unstructured = *obj.DeepCopy()

		// TODO(ezgidemirel): Refactor to reduce cyclomatic complexity.
		// Update only the status of the desired composite resource.
		if cd.Resource.GetAPIVersion() == observedComposite.Resource.GetAPIVersion() && cd.Resource.GetKind() == observedComposite.Resource.GetKind() {
			dst := make(map[string]any)
			if err := desiredComposite.Resource.GetValueInto("status", &dst); err != nil && !fieldpath.IsNotFound(err) {
				f.log.Info("cannot get desired composite status")
				response.Fatal(rsp, errors.Wrap(err, "cannot get desired composite status"))
				return rsp, nil
			}

			src := make(map[string]any)
			if err := cd.Resource.GetValueInto("status", &src); err != nil && !fieldpath.IsNotFound(err) {
				f.log.Info("cannot get templated composite status")
				response.Fatal(rsp, errors.Wrap(err, "cannot get templated composite status"))
				return rsp, nil
			}

			if err := mergo.Merge(&dst, src, mergo.WithOverride); err != nil {
				f.log.Info("cannot merge desired composite status")
				response.Fatal(rsp, errors.Wrap(err, "cannot merge desired composite status"))
				return rsp, nil
			}

			if err := fieldpath.Pave(desiredComposite.Resource.Object).SetValue("status", dst); err != nil {
				f.log.Info("cannot set desired composite status")
				response.Fatal(rsp, errors.Wrap(err, "cannot set desired composite status"))
				return rsp, nil
			}

			continue
		}

		// TODO(ezgidemirel): Refactor to reduce cyclomatic complexity.
		// Set composite resource's connection details.
		if cd.Resource.GetAPIVersion() == metaApiVersion {
			switch obj.GetKind() {
			case "CompositeConnectionDetails":
				con, _ := cd.Resource.GetStringObject("data")
				for k, v := range con {
					d, _ := base64.StdEncoding.DecodeString(v) //nolint:errcheck // k8s returns secret values encoded
					desiredComposite.ConnectionDetails[k] = d
				}
			default:
				f.log.Info("invalid kind %q for apiVersion %q - must be CompositeConnectionDetails", obj.GetKind(), metaApiVersion)
				response.Fatal(rsp, errors.Errorf("invalid kind %q for apiVersion %q - must be CompositeConnectionDetails", obj.GetKind(), metaApiVersion))
				return rsp, nil
			}

			continue
		}

		// TODO(ezgidemirel): Refactor to reduce cyclomatic complexity.
		// Set ready state.
		if v, found := cd.Resource.GetAnnotations()[annotationKeyReady]; found {
			if v != string(resource.ReadyTrue) && v != string(resource.ReadyUnspecified) && v != string(resource.ReadyFalse) {
				f.log.Info("invalid function input: invalid %q annotation value %q: must be True, False, or Unspecified", annotationKeyReady, v)
				response.Fatal(rsp, errors.Errorf("invalid function input: invalid %q annotation value %q: must be True, False, or Unspecified", annotationKeyReady, v))
				return rsp, nil
			}

			cd.Ready = resource.Ready(v)

			// Remove meta annotation.
			meta.RemoveAnnotations(cd.Resource, annotationKeyReady)
		}

		// Remove resource name annotation.
		meta.RemoveAnnotations(cd.Resource, annotationKeyCompositionResourceName)

		// Add resource to the desired composed resources map.
		name, found := obj.GetAnnotations()[annotationKeyCompositionResourceName]
		if !found {
			f.log.Info("%q template is missing required %q annotation", obj.GetKind(), annotationKeyCompositionResourceName)
			response.Fatal(rsp, errors.Errorf("%q template is missing required %q annotation", obj.GetKind(), annotationKeyCompositionResourceName))
			return rsp, nil
		}

		desiredComposed[resource.Name(name)] = cd

	}

	f.log.Info("desired composite resource", "desiredComposite:", desiredComposite)
	f.log.Info("constructed desired composed resources", "desiredComposed:", desiredComposed)

	if err := response.SetDesiredComposedResources(rsp, desiredComposed); err != nil {
		f.log.Info("cannot desired composed resources")
		response.Fatal(rsp, errors.Wrap(err, "cannot desired composed resources"))
		return rsp, nil
	}

	if err := response.SetDesiredCompositeResource(rsp, desiredComposite); err != nil {
		f.log.Info("cannot set desired composite resource")
		response.Fatal(rsp, errors.Wrap(err, "cannot set desired composite resource"))
		return rsp, nil
	}

	f.log.Info("Successfully composed desired resources", "source", in.Source, "count", len(objs))

	return rsp, nil
}

func convertToMap(req *fnv1beta1.RunFunctionRequest) (map[string]any, error) {
	jReq, err := protojson.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal request from proto to json")
	}

	var mReq map[string]any
	if err := json.Unmarshal(jReq, &mReq); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal json to map[string]any")
	}

	return mReq, nil
}
