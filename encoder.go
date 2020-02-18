package go3mf

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const defaultFloatPrecision = 6

type packagePart interface {
	io.Writer
	AddRelationship(Relationship)
}

type packageWriter interface {
	Create(name, contentType string) (packagePart, error)
	AddRelationship(Relationship)
	Close() error
}

// Marshaler is the interface implemented by objects
// that can marshal themselves into valid XML elements.
type Marshaler interface {
	Marshal3MF(x *XMLEncoder) error
}

// MarshalerAttr is the interface implemented by objects that can marshal
// themselves into valid XML attributes.
type MarshalerAttr interface {
	Marshal3MFAttr(*XMLEncoder) ([]xml.Attr, error)
}

// MarshalModel returns the XML encoding of m.
func MarshalModel(m *Model) ([]byte, error) {
	var b bytes.Buffer
	if err := new(Encoder).writeModel(newXMLEncoder(&b, defaultFloatPrecision), m); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// An Encoder writes Model data to an output stream.
//
// See the documentation for strconv.FormatFloat for details about the FloatPrecision behaviour.
type Encoder struct {
	FloatPrecision int
	w              packageWriter
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		FloatPrecision: defaultFloatPrecision,
		w:              newOpcWriter(w),
	}
}

// Encode writes the XML encoding of m to the stream.
func (e *Encoder) Encode(m *Model) error {
	if err := e.writeAttachements(m.Attachments); err != nil {
		return err
	}
	rootName := m.Path
	if rootName == "" {
		rootName = uriDefault3DModel
	}
	e.w.AddRelationship(Relationship{Type: RelTypeModel3D, Path: rootName})
	for _, r := range m.RootRelationships {
		e.w.AddRelationship(r)
	}

	w, err := e.w.Create(rootName, contentType3DModel)
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return err
	}
	enc := newXMLEncoder(w, e.FloatPrecision)
	enc.relationships = make([]Relationship, len(m.Relationships))
	copy(enc.relationships, m.Relationships)
	for path := range m.Childs {
		enc.AddRelationship(Relationship{Type: RelTypeModel3D, Path: path})
	}
	if err = e.writeModel(enc, m); err != nil {
		return err
	}
	for _, r := range enc.relationships {
		w.AddRelationship(r)
	}
	if err = e.writeChildModels(m); err != nil {
		return err
	}

	return e.w.Close()
}

func (e *Encoder) writeChildModels(m *Model) error {
	for path, child := range m.Childs {
		var (
			w   packagePart
			err error
		)
		if w, err = e.w.Create(path, contentType3DModel); err != nil {
			return err
		}
		if _, err = w.Write([]byte(xml.Header)); err != nil {
			return err
		}
		enc := newXMLEncoder(w, e.FloatPrecision)
		enc.relationships = child.Relationships
		if err = e.writeChildModel(enc, m, path); err != nil {
			return err
		}
		for _, r := range enc.relationships {
			w.AddRelationship(r)
		}
	}
	return nil
}

func (e *Encoder) writeAttachements(att []Attachment) error {
	for _, a := range att {
		w, err := e.w.Create(a.Path, a.ContentType)
		if err == nil {
			_, err = io.Copy(w, a.Stream)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) modelToken(x *XMLEncoder, m *Model, isRoot bool) (xml.StartElement, error) {
	attrs := []xml.Attr{
		{Name: xml.Name{Local: attrXmlns}, Value: ExtensionName},
		{Name: xml.Name{Local: attrUnit}, Value: m.Units.String()},
		{Name: xml.Name{Space: nsXML, Local: attrLang}, Value: m.Language},
	}
	if isRoot && m.Thumbnail != "" {
		x.AddRelationship(Relationship{Path: m.Thumbnail, Type: RelTypeThumbnail})
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: attrThumbnail}, Value: m.Thumbnail})
	}
	for _, a := range m.Namespaces {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Space: attrXmlns, Local: a.Local}, Value: a.Space})
	}
	if len(m.RequiredExtensions) != 0 {
		exts := make([]string, len(m.RequiredExtensions))
		for i, ns := range m.RequiredExtensions {
			for _, a := range m.Namespaces {
				if a.Space == ns {
					exts[i] = a.Local
					break
				}
			}
			if exts[i] == "" {
				return xml.StartElement{}, fmt.Errorf("go3mf: cannot encode model with undefined required extension '%s'", ns)
			}
		}
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: attrReqExt}, Value: strings.Join(exts, " ")})
	}
	tm := xml.StartElement{Name: xml.Name{Local: attrModel}, Attr: attrs}
	m.ExtensionAttr.encode(x, &tm)
	return tm, nil
}

func (e *Encoder) writeChildModel(x *XMLEncoder, m *Model, path string) error {
	tm, _ := e.modelToken(x, m, false) // error already checked before
	x.EncodeToken(tm)

	child := m.Childs[path]
	if err := e.writeResources(x, &child.Resources); err != nil {
		return err
	}

	xb := xml.StartElement{Name: xml.Name{Local: attrBuild}}
	x.EncodeToken(xb)
	x.EncodeToken(xb.End())
	child.Extension.encode(x)
	x.EncodeToken(tm.End())
	return x.Flush()
}

func (e *Encoder) writeModel(x *XMLEncoder, m *Model) error {
	tm, err := e.modelToken(x, m, true)
	if err != nil {
		return err
	}
	x.EncodeToken(tm)

	e.writeMetadata(x, m.Metadata)
	if err := e.writeResources(x, &m.Resources); err != nil {
		return err
	}
	e.writeBuild(x, m)
	m.Extension.encode(x)
	x.EncodeToken(tm.End())
	return x.Flush()
}

func (e *Encoder) writeMetadataGroup(x *XMLEncoder, m []Metadata) {
	xm := xml.StartElement{Name: xml.Name{Local: attrMetadataGroup}}
	x.EncodeToken(xm)
	e.writeMetadata(x, m)
	x.EncodeToken(xm.End())
}

func (e *Encoder) writeBuild(x *XMLEncoder, m *Model) {
	xb := xml.StartElement{Name: xml.Name{Local: attrBuild}}
	m.Build.ExtensionAttr.encode(x, &xb)
	x.EncodeToken(xb)
	x.SetAutoClose(true)
	for _, item := range m.Build.Items {
		xi := xml.StartElement{Name: xml.Name{Local: attrItem}, Attr: []xml.Attr{
			{Name: xml.Name{Local: attrObjectID}, Value: strconv.FormatUint(uint64(item.ObjectID), 10)},
		}}
		if item.HasTransform() {
			xi.Attr = append(xi.Attr, xml.Attr{
				Name: xml.Name{Local: attrTransform}, Value: item.Transform.String(),
			})
		}
		if item.PartNumber != "" {
			xi.Attr = append(xi.Attr, xml.Attr{
				Name: xml.Name{Local: attrPartNumber}, Value: item.PartNumber,
			})
		}
		item.ExtensionAttr.encode(x, &xi)
		if len(item.Metadata) != 0 {
			x.SetAutoClose(false)
			x.EncodeToken(xi)
			e.writeMetadataGroup(x, item.Metadata)
			x.EncodeToken(xi.End())
			x.SetAutoClose(true)
		} else {
			x.EncodeToken(xi)
		}
	}
	x.SetAutoClose(false)
	x.EncodeToken(xb.End())
}

func (e *Encoder) writeResources(x *XMLEncoder, rs *Resources) error {
	xt := xml.StartElement{Name: xml.Name{Local: attrResources}}
	x.EncodeToken(xt)
	for _, r := range rs.Assets {
		switch r := r.(type) {
		case *BaseMaterialsResource:
			e.writeBaseMaterial(x, r)
		case Marshaler:
			if err := r.Marshal3MF(x); err != nil {
				return err
			}
		}
		if err := x.Flush(); err != nil {
			return err
		}
	}

	for _, o := range rs.Objects {
		e.writeObject(x, o)
		if err := x.Flush(); err != nil {
			return err
		}
	}
	x.EncodeToken(xt.End())
	return nil
}

func (e *Encoder) writeMetadata(x *XMLEncoder, metadata []Metadata) {
	for _, md := range metadata {
		xn := xml.StartElement{Name: xml.Name{Local: attrMetadata}, Attr: []xml.Attr{
			{Name: xml.Name{Local: attrName}, Value: md.Name},
		}}
		if md.Preserve {
			xn.Attr = append(xn.Attr, xml.Attr{
				Name: xml.Name{Local: attrPreserve}, Value: strconv.FormatBool(md.Preserve),
			})
		}
		if md.Type != "" {
			xn.Attr = append(xn.Attr, xml.Attr{
				Name: xml.Name{Local: attrType}, Value: md.Type,
			})
		}
		x.EncodeToken(xn)
		x.EncodeToken(xml.CharData(md.Value))
		x.EncodeToken(xn.End())
	}
}

func (e *Encoder) writeObject(x *XMLEncoder, r *Object) {
	xo := xml.StartElement{Name: xml.Name{Local: attrObject}, Attr: []xml.Attr{
		{Name: xml.Name{Local: attrID}, Value: strconv.FormatUint(uint64(r.ID), 10)},
	}}
	if r.ObjectType != ObjectTypeModel {
		xo.Attr = append(xo.Attr, xml.Attr{Name: xml.Name{Local: attrType}, Value: r.ObjectType.String()})
	}
	if r.Thumbnail != "" {
		x.AddRelationship(Relationship{Path: r.Thumbnail, Type: RelTypeThumbnail})
		xo.Attr = append(xo.Attr, xml.Attr{Name: xml.Name{Local: attrThumbnail}, Value: r.Thumbnail})
	}
	if r.PartNumber != "" {
		xo.Attr = append(xo.Attr, xml.Attr{Name: xml.Name{Local: attrPartNumber}, Value: r.PartNumber})
	}
	if r.Name != "" {
		xo.Attr = append(xo.Attr, xml.Attr{Name: xml.Name{Local: attrName}, Value: r.Name})
	}
	if r.Mesh != nil {
		if r.DefaultPID != 0 {
			xo.Attr = append(xo.Attr, xml.Attr{
				Name: xml.Name{Local: attrPID}, Value: strconv.FormatUint(uint64(r.DefaultPID), 10),
			})
		}
		if r.DefaultPIndex != 0 {
			xo.Attr = append(xo.Attr, xml.Attr{
				Name: xml.Name{Local: attrPIndex}, Value: strconv.FormatUint(uint64(r.DefaultPIndex), 10),
			})
		}
	}
	r.ExtensionAttr.encode(x, &xo)
	x.EncodeToken(xo)

	if len(r.Metadata) != 0 {
		e.writeMetadataGroup(x, r.Metadata)
	}

	if r.Mesh != nil {
		e.writeMesh(x, r, r.Mesh)
	} else {
		e.writeComponents(x, r.Components)
	}
	x.EncodeToken(xo.End())
}

func (e *Encoder) writeComponents(x *XMLEncoder, comps []*Component) {
	xcs := xml.StartElement{Name: xml.Name{Local: attrComponents}}
	x.EncodeToken(xcs)
	x.SetAutoClose(true)
	for _, c := range comps {
		xt := xml.StartElement{
			Name: xml.Name{Local: attrComponent}, Attr: []xml.Attr{
				{Name: xml.Name{Local: attrObjectID}, Value: strconv.FormatUint(uint64(c.ObjectID), 10)},
			},
		}
		if c.HasTransform() {
			xt.Attr = append(xt.Attr, xml.Attr{Name: xml.Name{Local: attrTransform}, Value: c.Transform.String()})
		}
		c.ExtensionAttr.encode(x, &xt)
		x.EncodeToken(xt)
	}
	x.SetAutoClose(false)
	x.EncodeToken(xcs.End())
}

func (e *Encoder) writeMesh(x *XMLEncoder, r *Object, m *Mesh) {
	xm := xml.StartElement{Name: xml.Name{Local: attrMesh}}
	m.ExtensionAttr.encode(x, &xm)
	x.EncodeToken(xm)
	xvs := xml.StartElement{Name: xml.Name{Local: attrVertices}}
	x.EncodeToken(xvs)
	x.SetAutoClose(true)
	for _, v := range m.Nodes {
		x.EncodeToken(xml.StartElement{
			Name: xml.Name{Local: attrVertex},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: attrX}, Value: strconv.FormatFloat(float64(v.X()), 'f', x.FloatPresicion(), 32)},
				{Name: xml.Name{Local: attrY}, Value: strconv.FormatFloat(float64(v.Y()), 'f', x.FloatPresicion(), 32)},
				{Name: xml.Name{Local: attrZ}, Value: strconv.FormatFloat(float64(v.Z()), 'f', x.FloatPresicion(), 32)},
			},
		})
	}
	x.SetAutoClose(false)
	x.EncodeToken(xvs.End())

	xvt := xml.StartElement{Name: xml.Name{Local: attrTriangles}}
	x.EncodeToken(xvt)
	x.SetAutoClose(true)
	for _, v := range m.Faces {
		t := xml.StartElement{
			Name: xml.Name{Local: attrTriangle},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: attrV1}, Value: strconv.FormatUint(uint64(v.NodeIndices[0]), 10)},
				{Name: xml.Name{Local: attrV2}, Value: strconv.FormatUint(uint64(v.NodeIndices[1]), 10)},
				{Name: xml.Name{Local: attrV3}, Value: strconv.FormatUint(uint64(v.NodeIndices[2]), 10)},
			},
		}
		if v.PID != 0 {
			p1, p2, p3 := v.PIndex[0], v.PIndex[1], v.PIndex[2]
			if (p1 != p2) || (p1 != p3) {
				t.Attr = append(t.Attr,
					xml.Attr{Name: xml.Name{Local: attrPID}, Value: strconv.FormatUint(uint64(v.PID), 10)},
					xml.Attr{Name: xml.Name{Local: attrP1}, Value: strconv.FormatUint(uint64(p1), 10)},
					xml.Attr{Name: xml.Name{Local: attrP2}, Value: strconv.FormatUint(uint64(p2), 10)},
					xml.Attr{Name: xml.Name{Local: attrP3}, Value: strconv.FormatUint(uint64(p3), 10)},
				)
			} else if (v.PID != r.DefaultPID) || (p1 != r.DefaultPIndex) {
				t.Attr = append(t.Attr,
					xml.Attr{Name: xml.Name{Local: attrPID}, Value: strconv.FormatUint(uint64(v.PID), 10)},
					xml.Attr{Name: xml.Name{Local: attrP1}, Value: strconv.FormatUint(uint64(p1), 10)},
				)
			}
		}
		x.EncodeToken(t)
	}
	x.SetAutoClose(false)
	x.EncodeToken(xvt.End())
	m.Extension.encode(x)
	x.EncodeToken(xm.End())
}

func (e *Encoder) writeBaseMaterial(x *XMLEncoder, r *BaseMaterialsResource) {
	xt := xml.StartElement{Name: xml.Name{Local: attrBaseMaterials}, Attr: []xml.Attr{
		{Name: xml.Name{Local: attrID}, Value: strconv.FormatUint(uint64(r.ID), 10)},
	}}
	x.EncodeToken(xt)
	x.SetAutoClose(true)
	for _, ma := range r.Materials {
		x.EncodeToken(xml.StartElement{
			Name: xml.Name{Local: attrBase},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: attrName}, Value: ma.Name},
				{Name: xml.Name{Local: attrDisplayColor}, Value: FormatRGBA(ma.Color)},
			},
		})
	}
	x.SetAutoClose(false)
	x.EncodeToken(xt.End())
}