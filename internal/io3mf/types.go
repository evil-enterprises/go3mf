package io3mf

import "errors"

const (
	nsXML             = "http://www.w3.org/XML/1998/namespace"
	nsXMLNs           = "http://www.w3.org/2000/xmlns/"
	nsCoreSpec        = "http://schemas.microsoft.com/3dmanufacturing/core/2015/02"
	nsMaterialSpec    = "http://schemas.microsoft.com/3dmanufacturing/material/2015/02"
	nsProductionSpec  = "http://schemas.microsoft.com/3dmanufacturing/production/2015/06"
	nsBeamLatticeSpec = "http://schemas.microsoft.com/3dmanufacturing/beamlattice/2017/02"
	nsSliceSpec       = "http://schemas.microsoft.com/3dmanufacturing/slice/2015/07"
)

const (
	relTypeTexture3D = "http://schemas.microsoft.com/3dmanufacturing/2013/01/3dtexture"
	relTypeThumbnail = "http://schemas.openxmlformats.org/package/2006/relationships/metadata/thumbnail"
	relTypeModel3D   = "http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"
)

const (
	attrProdUUID  = "UUID"
	attrProdPath  = "path"
	attrObjectID  = "objectid"
	attrTransform = "transform"
	attrUnit      = "unit"
	attrReqExt    = "requiredextensions"
	attrLang      = "lang"
	attrResources = "resources"
	attrBuild     = "build"
)

// WarningLevel defines the level of a reader warning.
type WarningLevel int

const (
	// InvalidMandatoryValue happens when a mandatory value is invalid.
	InvalidMandatoryValue WarningLevel = iota
	// MissingMandatoryValue happens when a mandatory value is missing.
	MissingMandatoryValue
	// InvalidOptionalValue happens when an optional value is invalid.
	InvalidOptionalValue
)

// ErrUserAborted defines a user function abort.
var ErrUserAborted = errors.New("go3mf: the called function was aborted by the user")

// ReadError defines a error while reading a 3mf.
type ReadError struct {
	Level   WarningLevel
	Message string
}

func (e *ReadError) Error() string {
	return e.Message
}
