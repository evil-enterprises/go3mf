package slices

import (
	"math"

	"github.com/qmuntal/go3mf"
	specerr "github.com/qmuntal/go3mf/errors"
)

func validTransform(t go3mf.Matrix) bool {
	return t[2] == 0 && t[6] == 0 && t[8] == 0 && t[9] == 0 && t[10] == 1
}

func (ext *SliceStackInfo) Validate(m *go3mf.Model, path string, e interface{}) []error {
	var (
		obj *go3mf.Object
		ok  bool
	)
	if obj, ok = e.(*go3mf.Object); !ok {
		return nil
	}
	var errs []error
	res, _ := m.FindResources(path)
	if ext.SliceStackID == 0 {
		errs = append(errs, &specerr.MissingFieldError{Name: attrSliceRefID})
	} else if r, ok := res.FindAsset(ext.SliceStackID); ok {
		if r, ok := r.(*SliceStack); ok {
			if !validateBuildTransforms(m, path, obj.ID) {
				errs = append(errs, specerr.ErrSliceInvalidTranform)
			}
			if obj.ObjectType == go3mf.ObjectTypeModel || obj.ObjectType == go3mf.ObjectTypeSolidSupport {
				if !checkAllClosed(m, r) {
					errs = append(errs, specerr.ErrSlicePolygonNotClosed)
				}
			}
		} else {
			errs = append(errs, specerr.ErrNonSliceStack)
		}
	} else {
		errs = append(errs, specerr.ErrMissingResource)
	}
	if ext.SliceResolution == ResolutionLow {
		var extRequired bool
		for _, r := range m.RequiredExtensions {
			if r == ExtensionName {
				extRequired = true
				break
			}
		}
		if !extRequired {
			errs = append(errs, specerr.ErrSliceExtRequired)
		}
	}
	return errs
}

func (r *SliceStack) Validate(m *go3mf.Model, path string) []error {
	var errs []error
	if (len(r.Slices) != 0 && len(r.Refs) != 0) ||
		(len(r.Slices) == 0 && len(r.Refs) == 0) {
		errs = append(errs, specerr.ErrSlicesAndRefs)
	}
	errs = append(errs, r.validateRefs(m, path)...)
	return append(errs, r.validateSlices(m, path)...)
}

func (r *SliceStack) validateSlices(_ *go3mf.Model, path string) []error {
	var errs []error
	lastTopZ := float32(-math.MaxFloat32)
	for j, slice := range r.Slices {
		if slice.TopZ == 0 {
			errs = append(errs, specerr.NewIndexed(path, slice, j, &specerr.MissingFieldError{Name: attrZTop}))
		} else if slice.TopZ < r.BottomZ {
			errs = append(errs, specerr.NewIndexed(path, slice, j, specerr.ErrSliceSmallTopZ))
		}
		if slice.TopZ <= lastTopZ {
			errs = append(errs, specerr.NewIndexed(path, slice, j, specerr.ErrSliceNoMonotonic))
		}
		lastTopZ = slice.TopZ
		if len(slice.Polygons) == 0 && len(slice.Vertices) == 0 {
			continue
		}
		if len(slice.Vertices) < 2 {
			errs = append(errs, specerr.NewIndexed(path, slice, j, specerr.ErrSliceInsufficientVertices))
		}
		if len(slice.Polygons) == 0 {
			errs = append(errs, specerr.NewIndexed(path, slice, j, specerr.ErrSliceInsufficientPolygons))
		}
		var perrs []error
		for k, p := range slice.Polygons {
			if len(p.Segments) < 1 {
				perrs = append(perrs, specerr.NewIndexed(path, p, k, specerr.ErrSliceInsufficientSegments))
			}
		}
		for _, err := range perrs {
			errs = append(errs, specerr.NewIndexed(path, slice, j, err))
		}
	}
	return errs
}

func (r *SliceStack) validateRefs(m *go3mf.Model, path string) []error {
	var errs []error
	lastTopZ := float32(-math.MaxFloat32)
	for i, ref := range r.Refs {
		valid := true
		if ref.Path == "" {
			valid = false
			errs = append(errs, specerr.NewIndexed(path, ref, i, &specerr.MissingFieldError{Name: attrSlicePath}))
		} else if ref.Path == path {
			valid = false
			errs = append(errs, specerr.NewIndexed(path, ref, i, specerr.ErrSliceRefSamePart))
		}
		if ref.SliceStackID == 0 {
			valid = false
			errs = append(errs, specerr.NewIndexed(path, ref, i, &specerr.MissingFieldError{Name: attrSliceRefID}))
		}
		if !valid {
			continue
		}
		if st, ok := m.FindAsset(ref.Path, ref.SliceStackID); ok {
			if st, ok := st.(*SliceStack); ok {
				if len(st.Refs) != 0 {
					errs = append(errs, specerr.NewIndexed(path, ref, i, specerr.ErrSliceRefRef))
				}
				if len(st.Slices) > 0 && st.Slices[0].TopZ <= lastTopZ {
					errs = append(errs, specerr.NewIndexed(path, ref, i, specerr.ErrSliceNoMonotonic))
				}
				if len(st.Slices) > 0 {
					lastTopZ = st.Slices[len(st.Slices)-1].TopZ
				}
			} else {
				errs = append(errs, specerr.NewIndexed(path, ref, i, specerr.ErrNonSliceStack))
			}
		} else {
			errs = append(errs, specerr.NewIndexed(path, ref, i, specerr.ErrMissingResource))
		}
	}
	return errs
}

func isSliceStackClosed(r *SliceStack) bool {
	for _, slice := range r.Slices {
		for _, p := range slice.Polygons {
			if len(p.Segments) > 0 && p.StartV != p.Segments[len(p.Segments)-1].V2 {
				return false
			}
		}
	}
	return true
}

func checkAllClosed(m *go3mf.Model, r *SliceStack) bool {
	if !isSliceStackClosed(r) {
		return false
	}
	for _, ref := range r.Refs {
		if st, ok := m.FindAsset(ref.Path, ref.SliceStackID); ok {
			if st, ok := st.(*SliceStack); ok {
				if !isSliceStackClosed(st) {
					return false
				}
			}
		}
	}

	return true
}

func validateBuildTransforms(m *go3mf.Model, path string, id uint32) bool {
	for _, item := range m.Build.Items {
		if item.ObjectID == id && item.ObjectPath(path) == path {
			if item.HasTransform() && !validTransform(item.Transform) {
				return false
			}
		}
		if o, ok := m.FindObject(item.ObjectPath(path), item.ObjectID); ok {
			if !validateObjectTransforms(m, o, path, id) {
				return false
			}
		}
	}
	return true
}

func validateObjectTransforms(m *go3mf.Model, o *go3mf.Object, path string, id uint32) bool {
	for _, c := range o.Components {
		if c.ObjectID == id && c.ObjectPath(path) == path {
			if c.HasTransform() && !validTransform(c.Transform) {
				return false
			}
		}
		if c.ObjectID == o.ID && c.ObjectPath(path) == path { // avoid circular references
			break
		} else {
			if o1, ok := m.FindObject(c.ObjectPath(path), c.ObjectID); ok {
				if !validateObjectTransforms(m, o1, path, id) {
					return false
				}
			}
		}
	}
	return true
}
