// Copyright 2011 The win Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package win

import (
	"syscall"
	"unsafe"
)

type TomConstants uint32

const (
	TomFalse                           TomConstants = 0
	TomTrue                                         = -1
	TomUndefined                                    = -9999999
	TomToggle                                       = -9999998
	TomAutoColor                                    = -9999997
	TomDefault                                      = -9999996
	TomSuspend                                      = -9999995
	TomResume                                       = -9999994
	TomApplyNow                                     = 0
	TomApplyLater                                   = 1
	TomTrackParms                                   = 2
	TomCacheParms                                   = 3
	TomApplyTmp                                     = 4
	TomDisableSmartFont                             = 8
	TomEnableSmartFont                              = 9
	TomUsePoints                                    = 10
	TomUseTwips                                     = 11
	TomBackward                                     = 0xc0000001
	TomForward                                      = 0x3fffffff
	TomMove                                         = 0
	TomExtend                                       = 1
	TomNoSelection                                  = 0
	TomSelectionIP                                  = 1
	TomSelectionNormal                              = 2
	TomSelectionFrame                               = 3
	TomSelectionColumn                              = 4
	TomSelectionRow                                 = 5
	TomSelectionBlock                               = 6
	TomSelectionInlineShape                         = 7
	TomSelectionShape                               = 8
	TomSelStartActive                               = 1
	TomSelAtEOL                                     = 2
	TomSelOvertype                                  = 4
	TomSelActive                                    = 8
	TomSelReplace                                   = 16
	TomEnd                                          = 0
	TomStart                                        = 32
	TomCollapseEnd                                  = 0
	TomCollapseStart                                = 1
	TomClientCoord                                  = 256
	TomAllowOffClient                               = 512
	TomTransform                                    = 1024
	TomObjectArg                                    = 2048
	TomAtEnd                                        = 4096
	TomNone                                         = 0
	TomSingle                                       = 1
	TomWords                                        = 2
	TomDouble                                       = 3
	TomDotted                                       = 4
	TomDash                                         = 5
	TomDashDot                                      = 6
	TomDashDotDot                                   = 7
	TomWave                                         = 8
	TomThick                                        = 9
	TomHair                                         = 10
	TomDoubleWave                                   = 11
	TomHeavyWave                                    = 12
	TomLongDash                                     = 13
	TomThickDash                                    = 14
	TomThickDashDot                                 = 15
	TomThickDashDotDot                              = 16
	TomThickDotted                                  = 17
	TomThickLongDash                                = 18
	TomLineSpaceSingle                              = 0
	TomLineSpace1pt5                                = 1
	TomLineSpaceDouble                              = 2
	TomLineSpaceAtLeast                             = 3
	TomLineSpaceExactly                             = 4
	TomLineSpaceMultiple                            = 5
	TomLineSpacePercent                             = 6
	TomAlignLeft                                    = 0
	TomAlignCenter                                  = 1
	TomAlignRight                                   = 2
	TomAlignJustify                                 = 3
	TomAlignDecimal                                 = 3
	TomAlignBar                                     = 4
	TomDefaultTab                                   = 5
	TomAlignInterWord                               = 3
	TomAlignNewspaper                               = 4
	TomAlignInterLetter                             = 5
	TomAlignScaled                                  = 6
	TomSpaces                                       = 0
	TomDots                                         = 1
	TomDashes                                       = 2
	TomLines                                        = 3
	TomThickLines                                   = 4
	TomEquals                                       = 5
	TomTabBack                                      = -3
	TomTabNext                                      = -2
	TomTabHere                                      = -1
	TomListNone                                     = 0
	TomListBullet                                   = 1
	TomListNumberAsArabic                           = 2
	TomListNumberAsLCLetter                         = 3
	TomListNumberAsUCLetter                         = 4
	TomListNumberAsLCRoman                          = 5
	TomListNumberAsUCRoman                          = 6
	TomListNumberAsSequence                         = 7
	TomListNumberedCircle                           = 8
	TomListNumberedBlackCircleWingding              = 9
	TomListNumberedWhiteCircleWingding              = 10
	TomListNumberedArabicWide                       = 11
	TomListNumberedChS                              = 12
	TomListNumberedChT                              = 13
	TomListNumberedJpnChS                           = 14
	TomListNumberedJpnKor                           = 15
	TomListNumberedArabic1                          = 16
	TomListNumberedArabic2                          = 17
	TomListNumberedHebrew                           = 18
	TomListNumberedThaiAlpha                        = 19
	TomListNumberedThaiNum                          = 20
	TomListNumberedHindiAlpha                       = 21
	TomListNumberedHindiAlpha1                      = 22
	TomListNumberedHindiNum                         = 23
	TomListParentheses                              = 0x10000
	TomListPeriod                                   = 0x20000
	TomListPlain                                    = 0x30000
	TomListNoNumber                                 = 0x40000
	TomListMinus                                    = 0x80000
	TomIgnoreNumberStyle                            = 0x1000000
	TomParaStyleNormal                              = -1
	TomParaStyleHeading1                            = -2
	TomParaStyleHeading2                            = -3
	TomParaStyleHeading3                            = -4
	TomParaStyleHeading4                            = -5
	TomParaStyleHeading5                            = -6
	TomParaStyleHeading6                            = -7
	TomParaStyleHeading7                            = -8
	TomParaStyleHeading8                            = -9
	TomParaStyleHeading9                            = -10
	TomCharacter                                    = 1
	TomWord                                         = 2
	TomSentence                                     = 3
	TomParagraph                                    = 4
	TomLine                                         = 5
	TomStory                                        = 6
	TomScreen                                       = 7
	TomSection                                      = 8
	TomTableColumn                                  = 9
	TomColumn                                       = 9
	TomRow                                          = 10
	TomWindow                                       = 11
	TomCell                                         = 12
	TomCharFormat                                   = 13
	TomParaFormat                                   = 14
	TomTable                                        = 15
	TomObject                                       = 16
	TomPage                                         = 17
	TomHardParagraph                                = 18
	TomCluster                                      = 19
	TomInlineObject                                 = 20
	TomInlineObjectArg                              = 21
	TomLeafLine                                     = 22
	TomLayoutColumn                                 = 23
	TomProcessId                                    = 0x40000001
	TomMatchWord                                    = 2
	TomMatchCase                                    = 4
	TomMatchPattern                                 = 8
	TomUnknownStory                                 = 0
	TomMainTextStory                                = 1
	TomFootnotesStory                               = 2
	TomEndnotesStory                                = 3
	TomCommentsStory                                = 4
	TomTextFrameStory                               = 5
	TomEvenPagesHeaderStory                         = 6
	TomPrimaryHeaderStory                           = 7
	TomEvenPagesFooterStory                         = 8
	TomPrimaryFooterStory                           = 9
	TomFirstPageHeaderStory                         = 10
	TomFirstPageFooterStory                         = 11
	TomScratchStory                                 = 127
	TomFindStory                                    = 128
	TomReplaceStory                                 = 129
	TomStoryInactive                                = 0
	TomStoryActiveDisplay                           = 1
	TomStoryActiveUI                                = 2
	TomStoryActiveDisplayUI                         = 3
	TomNoAnimation                                  = 0
	TomLasVegasLights                               = 1
	TomBlinkingBackground                           = 2
	TomSparkleText                                  = 3
	TomMarchingBlackAnts                            = 4
	TomMarchingRedAnts                              = 5
	TomShimmer                                      = 6
	TomWipeDown                                     = 7
	TomWipeRight                                    = 8
	TomAnimationMax                                 = 8
	TomLowerCase                                    = 0
	TomUpperCase                                    = 1
	TomTitleCase                                    = 2
	TomSentenceCase                                 = 4
	TomToggleCase                                   = 5
	TomReadOnly                                     = 0x100
	TomShareDenyRead                                = 0x200
	TomShareDenyWrite                               = 0x400
	TomPasteFile                                    = 0x1000
	TomCreateNew                                    = 0x10
	TomCreateAlways                                 = 0x20
	TomOpenExisting                                 = 0x30
	TomOpenAlways                                   = 0x40
	TomTruncateExisting                             = 0x50
	TomRTF                                          = 0x1
	TomText                                         = 0x2
	TomHTML                                         = 0x3
	TomWordDocument                                 = 0x4
	TomBold                                         = 0x80000001
	TomItalic                                       = 0x80000002
	TomUnderline                                    = 0x80000004
	TomStrikeout                                    = 0x80000008
	TomProtected                                    = 0x80000010
	TomLink                                         = 0x80000020
	TomSmallCaps                                    = 0x80000040
	TomAllCaps                                      = 0x80000080
	TomHidden                                       = 0x80000100
	TomOutline                                      = 0x80000200
	TomShadow                                       = 0x80000400
	TomEmboss                                       = 0x80000800
	TomImprint                                      = 0x80001000
	TomDisabled                                     = 0x80002000
	TomRevised                                      = 0x80004000
	TomSubscriptCF                                  = 0x80010000
	TomSuperscriptCF                                = 0x80020000
	TomFontBound                                    = 0x80100000
	TomLinkProtected                                = 0x80800000
	TomInlineObjectStart                            = 0x81000000
	TomExtendedChar                                 = 0x82000000
	TomAutoBackColor                                = 0x84000000
	TomMathZoneNoBuildUp                            = 0x88000000
	TomMathZone                                     = 0x90000000
	TomMathZoneOrdinary                             = 0xa0000000
	TomAutoTextColor                                = 0xc0000000
	TomMathZoneDisplay                              = 0x40000
	TomParaEffectRTL                                = 0x1
	TomParaEffectKeep                               = 0x2
	TomParaEffectKeepNext                           = 0x4
	TomParaEffectPageBreakBefore                    = 0x8
	TomParaEffectNoLineNumber                       = 0x10
	TomParaEffectNoWidowControl                     = 0x20
	TomParaEffectDoNotHyphen                        = 0x40
	TomParaEffectSideBySide                         = 0x80
	TomParaEffectCollapsed                          = 0x100
	TomParaEffectOutlineLevel                       = 0x200
	TomParaEffectBox                                = 0x400
	TomParaEffectTableRowDelimiter                  = 0x1000
	TomParaEffectTable                              = 0x4000
	TomModWidthPairs                                = 0x1
	TomModWidthSpace                                = 0x2
	TomAutoSpaceAlpha                               = 0x4
	TomAutoSpaceNumeric                             = 0x8
	TomAutoSpaceParens                              = 0x10
	TomEmbeddedFont                                 = 0x20
	TomDoublestrike                                 = 0x40
	TomOverlapping                                  = 0x80
	TomNormalCaret                                  = 0
	TomKoreanBlockCaret                             = 0x1
	TomNullCaret                                    = 0x2
	TomIncludeInset                                 = 0x1
	TomUnicodeBiDi                                  = 0x1
	TomMathCFCheck                                  = 0x4
	TomUnlink                                       = 0x8
	TomUnhide                                       = 0x10
	TomCheckTextLimit                               = 0x20
	TomIgnoreCurrentFont                            = 0
	TomMatchCharRep                                 = 0x1
	TomMatchFontSignature                           = 0x2
	TomMatchAscii                                   = 0x4
	TomGetHeightOnly                                = 0x8
	TomMatchMathFont                                = 0x10
	TomCharset                                      = 0x80000000
	TomCharRepFromLcid                              = 0x40000000
	TomAnsi                                         = 0
	TomEastEurope                                   = 1
	TomCyrillic                                     = 2
	TomGreek                                        = 3
	TomTurkish                                      = 4
	TomHebrew                                       = 5
	TomArabic                                       = 6
	TomBaltic                                       = 7
	TomVietnamese                                   = 8
	TomDefaultCharRep                               = 9
	TomSymbol                                       = 10
	TomThai                                         = 11
	TomShiftJIS                                     = 12
	TomGB2312                                       = 13
	TomHangul                                       = 14
	TomBIG5                                         = 15
	TomPC437                                        = 16
	TomOEM                                          = 17
	TomMac                                          = 18
	TomArmenian                                     = 19
	TomSyriac                                       = 20
	TomThaana                                       = 21
	TomDevanagari                                   = 22
	TomBengali                                      = 23
	TomGurmukhi                                     = 24
	TomGujarati                                     = 25
	TomOriya                                        = 26
	TomTamil                                        = 27
	TomTelugu                                       = 28
	TomKannada                                      = 29
	TomMalayalam                                    = 30
	TomSinhala                                      = 31
	TomLao                                          = 32
	TomTibetan                                      = 33
	TomMyanmar                                      = 34
	TomGeorgian                                     = 35
	TomJamo                                         = 36
	TomEthiopic                                     = 37
	TomCherokee                                     = 38
	TomAboriginal                                   = 39
	TomOgham                                        = 40
	TomRunic                                        = 41
	TomKhmer                                        = 42
	TomMongolian                                    = 43
	TomBraille                                      = 44
	TomYi                                           = 45
	TomLimbu                                        = 46
	TomTaiLe                                        = 47
	TomNewTaiLue                                    = 48
	TomSylotiNagri                                  = 49
	TomKharoshthi                                   = 50
	TomKayahli                                      = 51
	TomUsymbol                                      = 52
	TomEmoji                                        = 53
	TomGlagolitic                                   = 54
	TomLisu                                         = 55
	TomVai                                          = 56
	TomNKo                                          = 57
	TomOsmanya                                      = 58
	TomPhagsPa                                      = 59
	TomGothic                                       = 60
	TomDeseret                                      = 61
	TomTifinagh                                     = 62
	TomCharRepMax                                   = 63
	TomRE10Mode                                     = 0x1
	TomUseAtFont                                    = 0x2
	TomTextFlowMask                                 = 0xc
	TomTextFlowES                                   = 0
	TomTextFlowSW                                   = 0x4
	TomTextFlowWN                                   = 0x8
	TomTextFlowNE                                   = 0xc
	TomNoIME                                        = 0x80000
	TomSelfIME                                      = 0x40000
	TomNoUpScroll                                   = 0x10000
	TomNoVpScroll                                   = 0x40000
	TomNoLink                                       = 0
	TomClientLink                                   = 1
	TomFriendlyLinkName                             = 2
	TomFriendlyLinkAddress                          = 3
	TomAutoLinkURL                                  = 4
	TomAutoLinkEmail                                = 5
	TomAutoLinkPhone                                = 6
	TomAutoLinkPath                                 = 7
	TomCompressNone                                 = 0
	TomCompressPunctuation                          = 1
	TomCompressPunctuationAndKana                   = 2
	TomCompressMax                                  = 2
	TomUnderlinePositionAuto                        = 0
	TomUnderlinePositionBelow                       = 1
	TomUnderlinePositionAbove                       = 2
	TomUnderlinePositionMax                         = 2
	TomFontAlignmentAuto                            = 0
	TomFontAlignmentTop                             = 1
	TomFontAlignmentBaseline                        = 2
	TomFontAlignmentBottom                          = 3
	TomFontAlignmentCenter                          = 4
	TomFontAlignmentMax                             = 4
	TomRubyBelow                                    = 0x80
	TomRubyAlignCenter                              = 0
	TomRubyAlign010                                 = 1
	TomRubyAlign121                                 = 2
	TomRubyAlignLeft                                = 3
	TomRubyAlignRight                               = 4
	TomLimitsDefault                                = 0
	TomLimitsUnderOver                              = 1
	TomLimitsSubSup                                 = 2
	TomUpperLimitAsSuperScript                      = 3
	TomLimitsOpposite                               = 4
	TomShowLLimPlaceHldr                            = 8
	TomShowULimPlaceHldr                            = 16
	TomDontGrowWithContent                          = 64
	TomGrowWithContent                              = 128
	TomSubSupAlign                                  = 1
	TomLimitAlignMask                               = 3
	TomLimitAlignCenter                             = 0
	TomLimitAlignLeft                               = 1
	TomLimitAlignRight                              = 2
	TomShowDegPlaceHldr                             = 8
	TomAlignDefault                                 = 0
	TomAlignMatchAscentDescent                      = 2
	TomMathVariant                                  = 0x20
	TomStyleDefault                                 = 0
	TomStyleScriptScriptCramped                     = 1
	TomStyleScriptScript                            = 2
	TomStyleScriptCramped                           = 3
	TomStyleScript                                  = 4
	TomStyleTextCramped                             = 5
	TomStyleText                                    = 6
	TomStyleDisplayCramped                          = 7
	TomStyleDisplay                                 = 8
	TomMathRelSize                                  = 0x40
	TomDecDecSize                                   = 0xfe
	TomDecSize                                      = 0xff
	TomIncSize                                      = (1 | TomMathRelSize)
	TomIncIncSize                                   = (2 | TomMathRelSize)
	TomGravityUI                                    = 0
	TomGravityBack                                  = 1
	TomGravityFore                                  = 2
	TomGravityIn                                    = 3
	TomGravityOut                                   = 4
	TomGravityBackward                              = 0x20000000
	TomGravityForward                               = 0x40000000
	TomAdjustCRLF                                   = 1
	TomUseCRLF                                      = 2
	TomTextize                                      = 4
	TomAllowFinalEOP                                = 8
	TomFoldMathAlpha                                = 16
	TomNoHidden                                     = 32
	TomIncludeNumbering                             = 64
	TomTranslateTableCell                           = 128
	TomNoMathZoneBrackets                           = 0x100
	TomConvertMathChar                              = 0x200
	TomNoUCGreekItalic                              = 0x400
	TomAllowMathBold                                = 0x800
	TomLanguageTag                                  = 0x1000
	TomConvertRTF                                   = 0x2000
	TomApplyRtfDocProps                             = 0x4000
	TomPhantomShow                                  = 1
	TomPhantomZeroWidth                             = 2
	TomPhantomZeroAscent                            = 4
	TomPhantomZeroDescent                           = 8
	TomPhantomTransparent                           = 16
	TomPhantomASmash                                = (TomPhantomShow | TomPhantomZeroAscent)
	TomPhantomDSmash                                = (TomPhantomShow | TomPhantomZeroDescent)
	TomPhantomHSmash                                = (TomPhantomShow | TomPhantomZeroWidth)
	TomPhantomSmash                                 = ((TomPhantomShow | TomPhantomZeroAscent) | TomPhantomZeroDescent)
	TomPhantomHorz                                  = (TomPhantomZeroAscent | TomPhantomZeroDescent)
	TomPhantomVert                                  = TomPhantomZeroWidth
	TomBoxHideTop                                   = 1
	TomBoxHideBottom                                = 2
	TomBoxHideLeft                                  = 4
	TomBoxHideRight                                 = 8
	TomBoxStrikeH                                   = 16
	TomBoxStrikeV                                   = 32
	TomBoxStrikeTLBR                                = 64
	TomBoxStrikeBLTR                                = 128
	TomBoxAlignCenter                               = 1
	TomSpaceMask                                    = 0x1c
	TomSpaceDefault                                 = 0
	TomSpaceUnary                                   = 4
	TomSpaceBinary                                  = 8
	TomSpaceRelational                              = 12
	TomSpaceSkip                                    = 16
	TomSpaceOrd                                     = 20
	TomSpaceDifferential                            = 24
	TomSizeText                                     = 32
	TomSizeScript                                   = 64
	TomSizeScriptScript                             = 96
	TomNoBreak                                      = 128
	TomTransparentForPositioning                    = 256
	TomTransparentForSpacing                        = 512
	TomStretchCharBelow                             = 0
	TomStretchCharAbove                             = 1
	TomStretchBaseBelow                             = 2
	TomStretchBaseAbove                             = 3
	TomMatrixAlignMask                              = 3
	TomMatrixAlignCenter                            = 0
	TomMatrixAlignTopRow                            = 1
	TomMatrixAlignBottomRow                         = 3
	TomShowMatPlaceHldr                             = 8
	TomEqArrayLayoutWidth                           = 1
	TomEqArrayAlignMask                             = 0xc
	TomEqArrayAlignCenter                           = 0
	TomEqArrayAlignTopRow                           = 4
	TomEqArrayAlignBottomRow                        = 0xc
	TomMathManualBreakMask                          = 0x7f
	TomMathBreakLeft                                = 0x7d
	TomMathBreakCenter                              = 0x7e
	TomMathBreakRight                               = 0x7f
	TomMathEqAlign                                  = 0x80
	TomMathArgShadingStart                          = 0x251
	TomMathArgShadingEnd                            = 0x252
	TomMathObjShadingStart                          = 0x253
	TomMathObjShadingEnd                            = 0x254
	TomFunctionTypeNone                             = 0
	TomFunctionTypeTakesArg                         = 1
	TomFunctionTypeTakesLim                         = 2
	TomFunctionTypeTakesLim2                        = 3
	TomFunctionTypeIsLim                            = 4
	TomMathParaAlignDefault                         = 0
	TomMathParaAlignCenterGroup                     = 1
	TomMathParaAlignCenter                          = 2
	TomMathParaAlignLeft                            = 3
	TomMathParaAlignRight                           = 4
	TomMathDispAlignMask                            = 3
	TomMathDispAlignCenterGroup                     = 0
	TomMathDispAlignCenter                          = 1
	TomMathDispAlignLeft                            = 2
	TomMathDispAlignRight                           = 3
	TomMathDispIntUnderOver                         = 4
	TomMathDispFracTeX                              = 8
	TomMathDispNaryGrow                             = 0x10
	TomMathDocEmptyArgMask                          = 0x60
	TomMathDocEmptyArgAuto                          = 0
	TomMathDocEmptyArgAlways                        = 0x20
	TomMathDocEmptyArgNever                         = 0x40
	TomMathDocSbSpOpUnchanged                       = 0x80
	TomMathDocDiffMask                              = 0x300
	TomMathDocDiffDefault                           = 0
	TomMathDocDiffUpright                           = 0x100
	TomMathDocDiffItalic                            = 0x200
	TomMathDocDiffOpenItalic                        = 0x300
	TomMathDispNarySubSup                           = 0x400
	TomMathDispDef                                  = 0x800
	TomMathEnableRtl                                = 0x1000
	TomMathBrkBinMask                               = 0x30000
	TomMathBrkBinBefore                             = 0
	TomMathBrkBinAfter                              = 0x10000
	TomMathBrkBinDup                                = 0x20000
	TomMathBrkBinSubMask                            = 0xc0000
	TomMathBrkBinSubMM                              = 0
	TomMathBrkBinSubPM                              = 0x40000
	TomMathBrkBinSubMP                              = 0x80000
	TomSelRange                                     = 0x255
	TomHstring                                      = 0x254
	TomFontPropTeXStyle                             = 0x33c
	TomFontPropAlign                                = 0x33d
	TomFontStretch                                  = 0x33e
	TomFontStyle                                    = 0x33f
	TomFontStyleUpright                             = 0
	TomFontStyleOblique                             = 1
	TomFontStyleItalic                              = 2
	TomFontStretchDefault                           = 0
	TomFontStretchUltraCondensed                    = 1
	TomFontStretchExtraCondensed                    = 2
	TomFontStretchCondensed                         = 3
	TomFontStretchSemiCondensed                     = 4
	TomFontStretchNormal                            = 5
	TomFontStretchSemiExpanded                      = 6
	TomFontStretchExpanded                          = 7
	TomFontStretchExtraExpanded                     = 8
	TomFontStretchUltraExpanded                     = 9
	TomFontWeightDefault                            = 0
	TomFontWeightThin                               = 100
	TomFontWeightExtraLight                         = 200
	TomFontWeightLight                              = 300
	TomFontWeightNormal                             = 400
	TomFontWeightRegular                            = 400
	TomFontWeightMedium                             = 500
	TomFontWeightSemiBold                           = 600
	TomFontWeightBold                               = 700
	TomFontWeightExtraBold                          = 800
	TomFontWeightBlack                              = 900
	TomFontWeightHeavy                              = 900
	TomFontWeightExtraBlack                         = 950
	TomParaPropMathAlign                            = 0x437
	TomDocMathBuild                                 = 0x80
	TomMathLMargin                                  = 0x81
	TomMathRMargin                                  = 0x82
	TomMathWrapIndent                               = 0x83
	TomMathWrapRight                                = 0x84
	TomMathPostSpace                                = 0x86
	TomMathPreSpace                                 = 0x85
	TomMathInterSpace                               = 0x87
	TomMathIntraSpace                               = 0x88
	TomCanCopy                                      = 0x89
	TomCanRedo                                      = 0x8a
	TomCanUndo                                      = 0x8b
	TomUndoLimit                                    = 0x8c
	TomDocAutoLink                                  = 0x8d
	TomEllipsisMode                                 = 0x8e
	TomEllipsisState                                = 0x8f
	TomEllipsisNone                                 = 0
	TomEllipsisEnd                                  = 1
	TomEllipsisWord                                 = 3
	TomEllipsisPresent                              = 1
	TomVTopCell                                     = 1
	TomVLowCell                                     = 2
	TomHStartCell                                   = 4
	TomHContCell                                    = 8
	TomRowUpdate                                    = 1
	TomRowApplyDefault                              = 0
	TomCellStructureChangeOnly                      = 1
	TomRowHeightActual                              = 0x80b
)

type OBJECTTYPE int32

const (
	TomSimpleText       OBJECTTYPE = 0
	TomRuby                        = (TomSimpleText + 1)
	TomHorzVert                    = (TomRuby + 1)
	TomWarichu                     = (TomHorzVert + 1)
	TomEq                          = 9
	TomMath                        = 10
	TomAccent                      = TomMath
	TomBox                         = (TomAccent + 1)
	TomBoxedFormula                = (TomBox + 1)
	TomBrackets                    = (TomBoxedFormula + 1)
	TomBracketsWithSeps            = (TomBrackets + 1)
	TomEquationArray               = (TomBracketsWithSeps + 1)
	TomFraction                    = (TomEquationArray + 1)
	TomFunctionApply               = (TomFraction + 1)
	TomLeftSubSup                  = (TomFunctionApply + 1)
	TomLowerLimit                  = (TomLeftSubSup + 1)
	TomMatrix                      = (TomLowerLimit + 1)
	TomNary                        = (TomMatrix + 1)
	TomOpChar                      = (TomNary + 1)
	TomOverbar                     = (TomOpChar + 1)
	TomPhanTom                     = (TomOverbar + 1)
	TomRadical                     = (TomPhanTom + 1)
	TomSlashedFraction             = (TomRadical + 1)
	TomStack                       = (TomSlashedFraction + 1)
	TomStretchStack                = (TomStack + 1)
	TomSubscript                   = (TomStretchStack + 1)
	TomSubSup                      = (TomSubscript + 1)
	TomSuperscript                 = (TomSubSup + 1)
	TomUnderbar                    = (TomSuperscript + 1)
	TomUpperLimit                  = (TomUnderbar + 1)
	TomObjectMax                   = TomUpperLimit
)

type ITextRangeVtbl struct {
	IDispatchVtbl
	GetText           uintptr
	SetText           uintptr
	GetChar           uintptr
	SetChar           uintptr
	GetDuplicate      uintptr
	GetFormattedText  uintptr
	SetFormattedText  uintptr
	GetStart          uintptr
	SetStart          uintptr
	GetEnd            uintptr
	SetEnd            uintptr
	GetFont           uintptr
	SetFont           uintptr
	GetPara           uintptr
	SetPara           uintptr
	GetStoryLength    uintptr
	GetStoryType      uintptr
	Collapse          uintptr
	Expand            uintptr
	GetIndex          uintptr
	SetIndex          uintptr
	SetRange          uintptr
	InRange           uintptr
	InStory           uintptr
	IsEqual           uintptr
	Select            uintptr
	StartOf           uintptr
	EndOf             uintptr
	Move              uintptr
	MoveStart         uintptr
	MoveEnd           uintptr
	MoveWhile         uintptr
	MoveStartWhile    uintptr
	MoveEndWhile      uintptr
	MoveUntil         uintptr
	MoveStartUntil    uintptr
	MoveEndUntil      uintptr
	FindText          uintptr
	FindTextStart     uintptr
	FindTextEnd       uintptr
	Delete            uintptr
	Cut               uintptr
	Copy              uintptr
	Paste             uintptr
	CanPaste          uintptr
	CanEdit           uintptr
	ChangeCase        uintptr
	GetPoint          uintptr
	SetPoint          uintptr
	ScrollIntoView    uintptr
	GetEmbeddedObject uintptr
}

type ITextRange struct {
	LpVtbl *ITextRangeVtbl
}

type ITextSelectionVtbl struct {
	ITextRangeVtbl
	GetFlags  uintptr
	SetFlags  uintptr
	GetType   uintptr
	MoveLeft  uintptr
	MoveRight uintptr
	MoveUp    uintptr
	MoveDown  uintptr
	HomeKey   uintptr
	EndKey    uintptr
	TypeText  uintptr
}

type ITextSelection struct {
	LpVtbl *ITextSelectionVtbl
}

type ITextDocumentVtbl struct {
	IDispatchVtbl
	GetName             uintptr
	GetSelection        uintptr
	GetStoryCount       uintptr
	GetStoryRanges      uintptr
	GetSaved            uintptr
	SetSaved            uintptr
	GetDefaultTabStop   uintptr
	SetDefaultTabStop   uintptr
	New                 uintptr
	Open                uintptr
	Save                uintptr
	Freeze              uintptr
	Unfreeze            uintptr
	BeginEditCollection uintptr
	EndEditCollection   uintptr
	Undo                uintptr
	Redo                uintptr
	Range               uintptr
	RangeFromPoint      uintptr
}

type ITextStoryRangesVtbl struct {
	IDispatchVtbl
	NewEnum  uintptr
	Item     uintptr
	GetCount uintptr
}

type ITextStoryRanges struct {
	LpVtbl *ITextStoryRangesVtbl
}

var (
	IID_ITextDocument = IID{0x8CC497C0, 0xA1DF, 0x11CE, [8]byte{0x80, 0x98, 0x00, 0xAA, 0x00, 0x47, 0xBE, 0x5D}}
)

type ITextDocument struct {
	LpVtbl *ITextDocumentVtbl
}

func (obj *ITextDocument) QueryInterface(riid REFIID, ppvObject *unsafe.Pointer) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.QueryInterface, 3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(riid)),
		uintptr(unsafe.Pointer(ppvObject)))
	return HRESULT(ret)
}

func (obj *ITextDocument) AddRef() uint32 {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.AddRef, 1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return uint32(ret)
}

func (obj *ITextDocument) Release() uint32 {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.Release, 1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return uint32(ret)
}

func (obj *ITextDocument) GetTypeInfoCount(pctinfo *uint32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.GetTypeInfoCount, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pctinfo)),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) GetTypeInfo(iTInfo uint32, lcid LCID, ppTInfo **ITypeInfo) HRESULT {
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.GetTypeInfo, 4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(iTInfo),
		uintptr(lcid),
		uintptr(unsafe.Pointer(ppTInfo)),
		0,
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) GetIDsOfNames(riid REFIID, rgszNames **uint16, cNames uint32, lcid LCID, rgDispId *DISPID) HRESULT {
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.GetIDsOfNames, 6,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(riid)),
		uintptr(unsafe.Pointer(rgszNames)),
		uintptr(cNames),
		uintptr(lcid),
		uintptr(unsafe.Pointer(rgDispId)))
	return HRESULT(ret)
}

func (obj *ITextDocument) Invoke(dispIdMember DISPID, riid REFIID, lcid LCID, wFlags uint16, pDispParams *DISPPARAMS, pVarResult *VARIANT, pExcepInfo *EXCEPINFO, puArgErr *uint32) HRESULT {
	ret, _, _ := syscall.Syscall9(obj.LpVtbl.Invoke, 9,
		uintptr(unsafe.Pointer(obj)),
		uintptr(dispIdMember),
		uintptr(unsafe.Pointer(riid)),
		uintptr(lcid),
		uintptr(wFlags),
		uintptr(unsafe.Pointer(pDispParams)),
		uintptr(unsafe.Pointer(pVarResult)),
		uintptr(unsafe.Pointer(pExcepInfo)),
		uintptr(unsafe.Pointer(puArgErr)))
	return HRESULT(ret)
}

func (obj *ITextDocument) GetName(pName **uint16 /*BSTR*/) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.GetName, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pName)),
		0)
	return HRESULT(ret)

}

func (obj *ITextDocument) GetSelection(ppSel **ITextSelection) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.GetSelection, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(ppSel)),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) GetStoryCount(pCount *int32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.GetStoryCount, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pCount)),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) GetStoryRanges(ppStories **ITextStoryRanges) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.GetStoryRanges, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(ppStories)),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) GetSaved(pValue *int32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.GetSaved, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pValue)),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) SetSaved(Value int32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.SetSaved, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(Value),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) GetDefaultTabStop(pValue *float32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.GetDefaultTabStop, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pValue)),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) SetDefaultTabStop(Value float32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.SetDefaultTabStop, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(Value),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) New() HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.New, 1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) Open(pVar *VARIANT, Flags int32, CodePage int32) HRESULT {
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.Open, 4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pVar)),
		uintptr(Flags),
		uintptr(CodePage),
		0,
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) Save(pVar *VARIANT, Flags int32, CodePage int32) HRESULT {
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.Save, 4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pVar)),
		uintptr(Flags),
		uintptr(CodePage),
		0,
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) Freeze(pCount *int32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.Freeze, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pCount)),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) Unfreeze(pCount *int32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.Freeze, 2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pCount)),
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) BeginEditCollection() HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.BeginEditCollection, 1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) EndEditCollection() HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.EndEditCollection, 1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) Undo(Count int32, pCount *int32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.Undo, 3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(Count),
		uintptr(unsafe.Pointer(pCount)))
	return HRESULT(ret)
}

func (obj *ITextDocument) Redo(Count int32, pCount *int32) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.Redo, 3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(Count),
		uintptr(unsafe.Pointer(pCount)))
	return HRESULT(ret)
}

func (obj *ITextDocument) Range(cpActive int32, cpAnchor int32, ppRange **ITextRange) HRESULT {
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.Range, 4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(cpActive),
		uintptr(cpAnchor),
		uintptr(unsafe.Pointer(ppRange)),
		0,
		0)
	return HRESULT(ret)
}

func (obj *ITextDocument) RangeFromPoint(x int32, y int32, ppRange **ITextRange) HRESULT {
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.RangeFromPoint, 4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(x),
		uintptr(y),
		uintptr(unsafe.Pointer(ppRange)),
		0,
		0)
	return HRESULT(ret)
}
