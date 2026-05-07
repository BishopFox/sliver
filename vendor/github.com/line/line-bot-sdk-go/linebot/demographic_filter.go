// Copyright 2020 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package linebot

import "encoding/json"

// DemographicFilter interface
type DemographicFilter interface {
	DemographicFilter()
}

// GenderType type
type GenderType string

// GenderType constants
const (
	GenderMale   GenderType = "male"
	GenderFemale GenderType = "female"
)

// GenderFilter type
type GenderFilter struct {
	Type    string       `json:"type"`
	Genders []GenderType `json:"oneOf"`
}

// NewGenderFilter function
func NewGenderFilter(genders ...GenderType) *GenderFilter {
	return &GenderFilter{
		Type:    "gender",
		Genders: genders,
	}
}

// DemographicFilter implements DemographicFilter interface
func (*GenderFilter) DemographicFilter() {}

// AgeType type
type AgeType string

// AgeType constants
const (
	AgeEmpty AgeType = ""
	Age15    AgeType = "age_15"
	Age20    AgeType = "age_20"
	Age25    AgeType = "age_25"
	Age30    AgeType = "age_30"
	Age35    AgeType = "age_35"
	Age40    AgeType = "age_40"
	Age45    AgeType = "age_45"
	Age50    AgeType = "age_50"
)

// AgeFilter type
type AgeFilter struct {
	Type string  `json:"type"`
	GTE  AgeType `json:"gte,omitempty"` // greater than or equal to
	LT   AgeType `json:"lt,omitempty"`  // less than
}

// NewAgeFilter function
func NewAgeFilter(gte, lt AgeType) *AgeFilter {
	return &AgeFilter{
		Type: "age",
		GTE:  gte,
		LT:   lt,
	}
}

// DemographicFilter implements DemographicFilter interface
func (*AgeFilter) DemographicFilter() {}

// AppType type
type AppType string

// AppType constants
const (
	AppTypeIOS     AppType = "ios"
	AppTypeAndroid AppType = "android"
)

// AppTypeFilter type
type AppTypeFilter struct {
	Type     string    `json:"type"`
	AppTypes []AppType `json:"oneOf"`
}

// NewAppTypeFilter function
func NewAppTypeFilter(appTypes ...AppType) *AppTypeFilter {
	return &AppTypeFilter{
		Type:     "appType",
		AppTypes: appTypes,
	}
}

// DemographicFilter implements DemographicFilter interface
func (*AppTypeFilter) DemographicFilter() {}

// AreaType type
type AreaType string

// AreaType constants
const (
	AreaJPHokkaido         AreaType = "jp_01"
	AreaJPAomori           AreaType = "jp_02"
	AreaJPIwate            AreaType = "jp_03"
	AreaJPMiyagi           AreaType = "jp_04"
	AreaJPAkita            AreaType = "jp_05"
	AreaJPYamagata         AreaType = "jp_06"
	AreaJPFukushima        AreaType = "jp_07"
	AreaJPIbaraki          AreaType = "jp_08"
	AreaJPTochigi          AreaType = "jp_09"
	AreaJPGunma            AreaType = "jp_10"
	AreaJPSaitama          AreaType = "jp_11"
	AreaJPChiba            AreaType = "jp_12"
	AreaJPTokyo            AreaType = "jp_13"
	AreaJPKanagawa         AreaType = "jp_14"
	AreaJPNiigata          AreaType = "jp_15"
	AreaJPToyama           AreaType = "jp_16"
	AreaJPIshikawa         AreaType = "jp_17"
	AreaJPFukui            AreaType = "jp_18"
	AreaJPYamanashi        AreaType = "jp_19"
	AreaJPNagano           AreaType = "jp_20"
	AreaJPGifu             AreaType = "jp_21"
	AreaJPShizuoka         AreaType = "jp_22"
	AreaJPAichi            AreaType = "jp_23"
	AreaJPMie              AreaType = "jp_24"
	AreaJPShiga            AreaType = "jp_25"
	AreaJPKyoto            AreaType = "jp_26"
	AreaJPOsaka            AreaType = "jp_27"
	AreaJPHyougo           AreaType = "jp_28"
	AreaJPNara             AreaType = "jp_29"
	AreaJPWakayama         AreaType = "jp_30"
	AreaJPTottori          AreaType = "jp_31"
	AreaJPShimane          AreaType = "jp_32"
	AreaJPOkayama          AreaType = "jp_33"
	AreaJPHiroshima        AreaType = "jp_34"
	AreaJPYamaguchi        AreaType = "jp_35"
	AreaJPTokushima        AreaType = "jp_36"
	AreaJPKagawa           AreaType = "jp_37"
	AreaJPEhime            AreaType = "jp_38"
	AreaJPKouchi           AreaType = "jp_39"
	AreaJPFukuoka          AreaType = "jp_40"
	AreaJPSaga             AreaType = "jp_41"
	AreaJPNagasaki         AreaType = "jp_42"
	AreaJPKumamoto         AreaType = "jp_43"
	AreaJPOita             AreaType = "jp_44"
	AreaJPMiyazaki         AreaType = "jp_45"
	AreaJPKagoshima        AreaType = "jp_46"
	AreaJPOkinawa          AreaType = "jp_47"
	AreaTWTaipeiCity       AreaType = "tw_01"
	AreaTWNewTaipeiCity    AreaType = "tw_02"
	AreaTWTaoyuanCity      AreaType = "tw_03"
	AreaTWTaichungCity     AreaType = "tw_04"
	AreaTWTainanCity       AreaType = "tw_05"
	AreaTWKaohsiungCity    AreaType = "tw_06"
	AreaTWKeelungCity      AreaType = "tw_07"
	AreaTWHsinchuCity      AreaType = "tw_08"
	AreaTWChiayiCity       AreaType = "tw_09"
	AreaTWHsinchuCounty    AreaType = "tw_10"
	AreaTWMiaoliCounty     AreaType = "tw_11"
	AreaTWChanghuaCounty   AreaType = "tw_12"
	AreaTWNantouCounty     AreaType = "tw_13"
	AreaTWYunlinCounty     AreaType = "tw_14"
	AreaTWChiayiCounty     AreaType = "tw_15"
	AreaTWPingtungCounty   AreaType = "tw_16"
	AreaTWYilanCounty      AreaType = "tw_17"
	AreaTWHualienCounty    AreaType = "tw_18"
	AreaTWTaitungCounty    AreaType = "tw_19"
	AreaTWPenghuCounty     AreaType = "tw_20"
	AreaTWKinmenCounty     AreaType = "tw_21"
	AreaTWLienchiangCounty AreaType = "tw_22"
	AreaTHBangkok          AreaType = "th_01"
	AreaTHPattaya          AreaType = "th_02"
	AreaTHNorthern         AreaType = "th_03"
	AreaTHCentral          AreaType = "th_04"
	AreaTHSouthern         AreaType = "th_05"
	AreaTHEastern          AreaType = "th_06"
	AreaTHNorthEastern     AreaType = "th_07"
	AreaTHWestern          AreaType = "th_08"
	AreaIDBali             AreaType = "id_01"
	AreaIDBandung          AreaType = "id_02"
	AreaIDBanjarmasin      AreaType = "id_03"
	AreaIDJabodetabek      AreaType = "id_04"
	AreaIDLainnya          AreaType = "id_05"
	AreaIDMakassar         AreaType = "id_06"
	AreaIDMedan            AreaType = "id_07"
	AreaIDPalembang        AreaType = "id_08"
	AreaIDSamarinda        AreaType = "id_09"
	AreaIDSemarang         AreaType = "id_10"
	AreaIDSurabaya         AreaType = "id_11"
	AreaIDYogyakarta       AreaType = "id_12"
)

// AreaFilter type
type AreaFilter struct {
	Type  string     `json:"type"`
	Areas []AreaType `json:"oneOf"`
}

// NewAreaFilter function
func NewAreaFilter(areaTypes ...AreaType) *AreaFilter {
	return &AreaFilter{
		Type:  "area",
		Areas: areaTypes,
	}
}

// DemographicFilter implements DemographicFilter interface
func (*AreaFilter) DemographicFilter() {}

// PeriodType type
type PeriodType string

// PeriodType constants
const (
	PeriodEmpty  PeriodType = ""
	PeriodDay7   PeriodType = "day_7"
	PeriodDay30  PeriodType = "day_30"
	PeriodDay90  PeriodType = "day_90"
	PeriodDay180 PeriodType = "day_180"
	PeriodDay365 PeriodType = "day_365"
)

// SubscriptionPeriodFilter type
type SubscriptionPeriodFilter struct {
	Type string     `json:"type"`
	GTE  PeriodType `json:"gte,omitempty"` // greater than or equal to
	LT   PeriodType `json:"lt,omitempty"`  // less than
}

// NewSubscriptionPeriodFilter function
func NewSubscriptionPeriodFilter(gte, lt PeriodType) *SubscriptionPeriodFilter {
	return &SubscriptionPeriodFilter{
		Type: "subscriptionPeriod",
		GTE:  gte,
		LT:   lt,
	}
}

// DemographicFilter implements DemographicFilter interface
func (*SubscriptionPeriodFilter) DemographicFilter() {}

// DemographicFilterOperator struct
type DemographicFilterOperator struct {
	ConditionAnd []DemographicFilter `json:"and,omitempty"`
	ConditionOr  []DemographicFilter `json:"or,omitempty"`
	ConditionNot DemographicFilter   `json:"not,omitempty"`
}

// DemographicFilterOperatorAnd method
func DemographicFilterOperatorAnd(conditions ...DemographicFilter) *DemographicFilterOperator {
	return &DemographicFilterOperator{
		ConditionAnd: conditions,
	}
}

// DemographicFilterOperatorOr method
func DemographicFilterOperatorOr(conditions ...DemographicFilter) *DemographicFilterOperator {
	return &DemographicFilterOperator{
		ConditionOr: conditions,
	}
}

// DemographicFilterOperatorNot method
func DemographicFilterOperatorNot(condition DemographicFilter) *DemographicFilterOperator {
	return &DemographicFilterOperator{
		ConditionNot: condition,
	}
}

// MarshalJSON method of DemographicFilterOperator
func (o *DemographicFilterOperator) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type         string              `json:"type"`
		ConditionAnd []DemographicFilter `json:"and,omitempty"`
		ConditionOr  []DemographicFilter `json:"or,omitempty"`
		ConditionNot DemographicFilter   `json:"not,omitempty"`
	}{
		Type:         "operator",
		ConditionAnd: o.ConditionAnd,
		ConditionOr:  o.ConditionOr,
		ConditionNot: o.ConditionNot,
	})
}

// DemographicFilter implements DemographicFilter interface
func (*DemographicFilterOperator) DemographicFilter() {}
