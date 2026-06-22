package printer

import (
	"regexp"
	"strings"
)

const (
	DeviceClassLabel      = "label"
	DeviceClassReceipt    = "receipt"
	DeviceClassDocument   = "document"
	DeviceClassColorLabel = "color-label"
	DeviceClassUnknown    = "unknown"

	MediaKindThermalLabel = "thermal-label"
	MediaKindReceiptRoll  = "receipt-roll"
	MediaKindA4Document   = "a4-document"
	MediaKindColorLabel   = "color-roll-label"
	MediaKindUnknown      = "unknown"

	CommandSetTSPL          = "tspl"
	CommandSetZPL           = "zpl"
	CommandSetEPL           = "epl"
	CommandSetDPL           = "dpl"
	CommandSetESCPOS        = "escpos"
	CommandSetImage         = "image"
	CommandSetBrotherRaster = "brother-raster"
	CommandSetBrotherPTouch = "brother-ptouch"
	CommandSetEpsonESCLabel = "epson-esclabel"
	CommandSetGodexEZPL     = "godex-ezpl"
	CommandSetSatoSBPL      = "sato-sbpl"
	CommandSetToshibaTPCL   = "toshiba-tpcl"
	CommandSetCabJScript    = "cab-jscript"
	CommandSetHoneywellDP   = "honeywell-dp"
	CommandSetUnknown       = "unknown"

	LabelLanguageTSPL    = "tspl"
	LabelLanguageZPL     = "zpl"
	LabelLanguageImage   = "image"
	LabelLanguageNone    = "none"
	LabelLanguageUnknown = "unknown"

	JobTypeRawValue   = "raw"
	JobTypeImageValue = "image"
	JobTypePDFValue   = "pdf"
	JobTypeNone       = "none"

	CapabilitySourceBuiltinModelRule = "builtin-model-rule"
	CapabilitySourceVendorRule       = "vendor-family-rule"
	CapabilitySourceFallback         = "fallback"

	CapabilityConfidenceHigh   = "high"
	CapabilityConfidenceMedium = "medium"
	CapabilityConfidenceLow    = "low"
)

// PrinterCapability describes what kind of printer this is and which print
// payload OpenPrintHub should prefer for browser-triggered label workflows.
type PrinterCapability struct {
	DeviceClass            string   `json:"device_class,omitempty"`
	MediaKind              string   `json:"media_kind,omitempty"`
	CommandSets            []string `json:"command_sets,omitempty"`
	SupportedJobTypes      []string `json:"supported_job_types,omitempty"`
	PreferredLabelLanguage string   `json:"preferred_label_language,omitempty"`
	PreferredJobType       string   `json:"preferred_job_type,omitempty"`
	DPI                    int      `json:"dpi,omitempty"`
	CapabilitySource       string   `json:"capability_source,omitempty"`
	CapabilityConfidence   string   `json:"capability_confidence,omitempty"`
}

type capabilityRule struct {
	pattern    *regexp.Regexp
	capability PrinterCapability
}

var capabilityRules = []capabilityRule{
	// Negative / exclusion rules first. These devices must not be offered as
	// medication-label printers even when their vendor also sells label models.
	capabilityRuleFromPattern(`(?i)\b(hp|hewlett).*(laserjet|officejet|deskjet|mfp)`, documentCapability(CapabilitySourceVendorRule)),
	capabilityRuleFromPattern(`(?i)brother.*(mfc|dcp|hl)`, documentCapability(CapabilitySourceVendorRule)),
	capabilityRuleFromPattern(`(?i)(canon|epson).*(pixma|ecotank|workforce|office)`, documentCapability(CapabilitySourceVendorRule)),
	capabilityRuleFromPattern(`(?i)xprinter.*(xp[-_ ]?n|xp[-_ ]?q|receipt|58|80)`, receiptCapability(CapabilitySourceBuiltinModelRule)),
	capabilityRuleFromPattern(`(?i)bixolon.*(srp[-_ ]?3|receipt|kitchen)`, receiptCapability(CapabilitySourceBuiltinModelRule)),
	capabilityRuleFromPattern(`(?i)(tm[-_ ]?t|tm[-_ ]?m|receipt|kitchen|srp[-_ ]?3|tsp100|tsp650|mc[-_ ]?print)`, receiptCapability(CapabilitySourceVendorRule)),
	capabilityRuleFromPattern(`(?i)epson.*(colorworks|cw[-_ ]?c|c40|c60|c65|c80)`, colorLabelCapability()),

	// Label printers with native TSPL / TSPL-EZD.
	capabilityRuleFromPattern(`(?i)xprinter.*xp[-_ ]?(420b|421b|410b|450b|470b|p441b)`, tsplCapability(CapabilitySourceBuiltinModelRule)),
	capabilityRuleFromPattern(`(?i)\btsc\b|ttp[-_ ]?(225|244|247)|tdp[-_ ]?225|te[23][01]0|tx[23]00|tc2|da[23]|me240|mx340|mh240`, tsplCapability(CapabilitySourceVendorRule)),

	// Label printers with native or credible ZPL / ZPL II emulation.
	capabilityRuleFromPattern(`(?i)zebra|zdesigner|zd[246]\d+|zt\d+|gk420|gx4[23]0|gc420|gt800|220xi`, zplCapability(CapabilitySourceVendorRule)),
	capabilityRuleFromPattern(`(?i)brother.*td[-_ ]?(44|45|4[0-9]{3}|4t)|brother.*\b(tj|rj)\b`, brotherTDCapability()),
	capabilityRuleFromPattern(`(?i)bixolon.*(xd3|xd5|slp[-_ ]?(tx|dx)|srp[-_ ]?e770)`, bixolonLabelCapability()),
	capabilityRuleFromPattern(`(?i)godex|ge3[03]0|g5[03]0|dt4x|rt2[03]0`, godexCapability()),
	capabilityRuleFromPattern(`(?i)honeywell.*(pc42|pm42|pm45|pd45|px940)`, honeywellCapability()),
	capabilityRuleFromPattern(`(?i)toshiba.*(b[-_ ]?ex|b[-_ ]?sx|b[-_ ]?852)`, toshibaCapability()),
	capabilityRuleFromPattern(`(?i)sato.*(ws2|ws4|cl4nx|cl6nx|ct4|sa4)`, satoCapability()),
	capabilityRuleFromPattern(`(?i)\bcab\b|eos2|eos5|squix|xc q|xd q`, cabCapability()),

	// Driver/raster label printers.
	capabilityRuleFromPattern(`(?i)brother.*ql[-_ ]?(8|11)`, brotherQLCapability()),
	capabilityRuleFromPattern(`(?i)dymo|labelwriter`, imageLabelCapability(CapabilitySourceVendorRule, 300)),
}

func capabilityRuleFromPattern(pattern string, capability PrinterCapability) capabilityRule {
	return capabilityRule{
		pattern:    regexp.MustCompile(pattern),
		capability: capability,
	}
}

// DetectCapabilities returns the best-known printer capability for an OS
// printer. It intentionally relies on conservative model-family rules: unknown
// printers never default to raw command payloads.
func DetectCapabilities(id, name, description string) PrinterCapability {
	searchText := strings.Join([]string{id, name, description}, " ")
	for _, rule := range capabilityRules {
		if rule.pattern.MatchString(searchText) {
			return rule.capability
		}
	}
	return unknownCapability()
}

// ApplyCapabilities enriches PrinterInfo while preserving all existing fields.
func ApplyCapabilities(info PrinterInfo) PrinterInfo {
	capability := DetectCapabilities(info.ID, info.Name, info.Description)
	info.DeviceClass = capability.DeviceClass
	info.MediaKind = capability.MediaKind
	info.CommandSets = capability.CommandSets
	info.SupportedJobTypes = capability.SupportedJobTypes
	info.PreferredLabelLanguage = capability.PreferredLabelLanguage
	info.PreferredJobType = capability.PreferredJobType
	info.DPI = capability.DPI
	info.CapabilitySource = capability.CapabilitySource
	info.CapabilityConfidence = capability.CapabilityConfidence
	return info
}

func tsplCapability(source string) PrinterCapability {
	return PrinterCapability{
		DeviceClass:            DeviceClassLabel,
		MediaKind:              MediaKindThermalLabel,
		CommandSets:            []string{CommandSetTSPL, CommandSetZPL, CommandSetEPL, CommandSetDPL},
		SupportedJobTypes:      []string{JobTypeRawValue, JobTypeImageValue},
		PreferredLabelLanguage: LabelLanguageTSPL,
		PreferredJobType:       JobTypeRawValue,
		DPI:                    203,
		CapabilitySource:       source,
		CapabilityConfidence:   CapabilityConfidenceHigh,
	}
}

func zplCapability(source string) PrinterCapability {
	return PrinterCapability{
		DeviceClass:            DeviceClassLabel,
		MediaKind:              MediaKindThermalLabel,
		CommandSets:            []string{CommandSetZPL, CommandSetEPL},
		SupportedJobTypes:      []string{JobTypeRawValue, JobTypeImageValue},
		PreferredLabelLanguage: LabelLanguageZPL,
		PreferredJobType:       JobTypeRawValue,
		DPI:                    203,
		CapabilitySource:       source,
		CapabilityConfidence:   CapabilityConfidenceHigh,
	}
}

func brotherTDCapability() PrinterCapability {
	capability := zplCapability(CapabilitySourceBuiltinModelRule)
	capability.CommandSets = []string{CommandSetZPL, CommandSetBrotherRaster, CommandSetBrotherPTouch}
	return capability
}

func bixolonLabelCapability() PrinterCapability {
	capability := zplCapability(CapabilitySourceBuiltinModelRule)
	capability.CommandSets = []string{CommandSetZPL, CommandSetEPL, "slcs"}
	return capability
}

func godexCapability() PrinterCapability {
	capability := zplCapability(CapabilitySourceBuiltinModelRule)
	capability.CommandSets = []string{CommandSetZPL, CommandSetEPL, CommandSetGodexEZPL, CommandSetDPL}
	return capability
}

func honeywellCapability() PrinterCapability {
	capability := zplCapability(CapabilitySourceBuiltinModelRule)
	capability.CommandSets = []string{CommandSetZPL, CommandSetEPL, CommandSetDPL, CommandSetHoneywellDP}
	return capability
}

func toshibaCapability() PrinterCapability {
	capability := zplCapability(CapabilitySourceBuiltinModelRule)
	capability.CommandSets = []string{CommandSetZPL, CommandSetDPL, CommandSetToshibaTPCL}
	capability.CapabilityConfidence = CapabilityConfidenceMedium
	return capability
}

func satoCapability() PrinterCapability {
	capability := zplCapability(CapabilitySourceBuiltinModelRule)
	capability.CommandSets = []string{CommandSetZPL, CommandSetEPL, CommandSetDPL, CommandSetSatoSBPL}
	capability.CapabilityConfidence = CapabilityConfidenceMedium
	return capability
}

func cabCapability() PrinterCapability {
	capability := zplCapability(CapabilitySourceBuiltinModelRule)
	capability.CommandSets = []string{CommandSetZPL, CommandSetCabJScript}
	capability.CapabilityConfidence = CapabilityConfidenceMedium
	return capability
}

func imageLabelCapability(source string, dpi int) PrinterCapability {
	return PrinterCapability{
		DeviceClass:            DeviceClassLabel,
		MediaKind:              MediaKindThermalLabel,
		CommandSets:            []string{CommandSetImage},
		SupportedJobTypes:      []string{JobTypeImageValue},
		PreferredLabelLanguage: LabelLanguageImage,
		PreferredJobType:       JobTypeImageValue,
		DPI:                    dpi,
		CapabilitySource:       source,
		CapabilityConfidence:   CapabilityConfidenceHigh,
	}
}

func brotherQLCapability() PrinterCapability {
	capability := imageLabelCapability(CapabilitySourceBuiltinModelRule, 300)
	capability.CommandSets = []string{CommandSetImage, CommandSetBrotherRaster, CommandSetBrotherPTouch}
	return capability
}

func documentCapability(source string) PrinterCapability {
	return PrinterCapability{
		DeviceClass:            DeviceClassDocument,
		MediaKind:              MediaKindA4Document,
		CommandSets:            nil,
		SupportedJobTypes:      []string{JobTypePDFValue, JobTypeImageValue},
		PreferredLabelLanguage: LabelLanguageNone,
		PreferredJobType:       JobTypeNone,
		CapabilitySource:       source,
		CapabilityConfidence:   CapabilityConfidenceHigh,
	}
}

func receiptCapability(source string) PrinterCapability {
	return PrinterCapability{
		DeviceClass:            DeviceClassReceipt,
		MediaKind:              MediaKindReceiptRoll,
		CommandSets:            []string{CommandSetESCPOS},
		SupportedJobTypes:      []string{JobTypeRawValue},
		PreferredLabelLanguage: LabelLanguageNone,
		PreferredJobType:       JobTypeNone,
		DPI:                    203,
		CapabilitySource:       source,
		CapabilityConfidence:   CapabilityConfidenceHigh,
	}
}

func colorLabelCapability() PrinterCapability {
	return PrinterCapability{
		DeviceClass:            DeviceClassColorLabel,
		MediaKind:              MediaKindColorLabel,
		CommandSets:            []string{CommandSetEpsonESCLabel},
		SupportedJobTypes:      []string{JobTypeImageValue, JobTypePDFValue},
		PreferredLabelLanguage: LabelLanguageNone,
		PreferredJobType:       JobTypeNone,
		CapabilitySource:       CapabilitySourceVendorRule,
		CapabilityConfidence:   CapabilityConfidenceMedium,
	}
}

func unknownCapability() PrinterCapability {
	return PrinterCapability{
		DeviceClass:            DeviceClassUnknown,
		MediaKind:              MediaKindUnknown,
		CommandSets:            []string{CommandSetUnknown},
		SupportedJobTypes:      []string{JobTypeImageValue},
		PreferredLabelLanguage: LabelLanguageImage,
		PreferredJobType:       JobTypeImageValue,
		CapabilitySource:       CapabilitySourceFallback,
		CapabilityConfidence:   CapabilityConfidenceLow,
	}
}
