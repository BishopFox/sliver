/*
 *
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

package ethtool

import (
	"golang.org/x/sys/unix"
)

// Updated supportedCapabilities including modes from ethtool.h enum ethtool_link_mode_bit_indices
var supportedCapabilities = []struct {
	name  string
	mask  uint64 // Use uint64 to accommodate indices > 31
	speed uint64 // Speed in bps, 0 for non-speed modes
}{
	// Existing entries (reordered slightly by bit index for clarity)
	{"10baseT_Half", unix.ETHTOOL_LINK_MODE_10baseT_Half_BIT, 10_000_000},        // 0
	{"10baseT_Full", unix.ETHTOOL_LINK_MODE_10baseT_Full_BIT, 10_000_000},        // 1
	{"100baseT_Half", unix.ETHTOOL_LINK_MODE_100baseT_Half_BIT, 100_000_000},     // 2
	{"100baseT_Full", unix.ETHTOOL_LINK_MODE_100baseT_Full_BIT, 100_000_000},     // 3
	{"1000baseT_Half", unix.ETHTOOL_LINK_MODE_1000baseT_Half_BIT, 1_000_000_000}, // 4
	{"1000baseT_Full", unix.ETHTOOL_LINK_MODE_1000baseT_Full_BIT, 1_000_000_000}, // 5
	// Newly added or re-confirmed based on full enum
	{"Autoneg", unix.ETHTOOL_LINK_MODE_Autoneg_BIT, 0},                                    // 6
	{"TP", unix.ETHTOOL_LINK_MODE_TP_BIT, 0},                                              // 7 (Twisted Pair port)
	{"AUI", unix.ETHTOOL_LINK_MODE_AUI_BIT, 0},                                            // 8 (AUI port)
	{"MII", unix.ETHTOOL_LINK_MODE_MII_BIT, 0},                                            // 9 (MII port)
	{"FIBRE", unix.ETHTOOL_LINK_MODE_FIBRE_BIT, 0},                                        // 10 (FIBRE port)
	{"BNC", unix.ETHTOOL_LINK_MODE_BNC_BIT, 0},                                            // 11 (BNC port)
	{"10000baseT_Full", unix.ETHTOOL_LINK_MODE_10000baseT_Full_BIT, 10_000_000_000},       // 12
	{"Pause", unix.ETHTOOL_LINK_MODE_Pause_BIT, 0},                                        // 13
	{"Asym_Pause", unix.ETHTOOL_LINK_MODE_Asym_Pause_BIT, 0},                              // 14
	{"2500baseX_Full", unix.ETHTOOL_LINK_MODE_2500baseX_Full_BIT, 2_500_000_000},          // 15
	{"Backplane", unix.ETHTOOL_LINK_MODE_Backplane_BIT, 0},                                // 16 (Backplane port)
	{"1000baseKX_Full", unix.ETHTOOL_LINK_MODE_1000baseKX_Full_BIT, 1_000_000_000},        // 17
	{"10000baseKX4_Full", unix.ETHTOOL_LINK_MODE_10000baseKX4_Full_BIT, 10_000_000_000},   // 18
	{"10000baseKR_Full", unix.ETHTOOL_LINK_MODE_10000baseKR_Full_BIT, 10_000_000_000},     // 19
	{"10000baseR_FEC", unix.ETHTOOL_LINK_MODE_10000baseR_FEC_BIT, 10_000_000_000},         // 20
	{"20000baseMLD2_Full", unix.ETHTOOL_LINK_MODE_20000baseMLD2_Full_BIT, 20_000_000_000}, // 21
	{"20000baseKR2_Full", unix.ETHTOOL_LINK_MODE_20000baseKR2_Full_BIT, 20_000_000_000},   // 22
	{"40000baseKR4_Full", unix.ETHTOOL_LINK_MODE_40000baseKR4_Full_BIT, 40_000_000_000},   // 23
	{"40000baseCR4_Full", unix.ETHTOOL_LINK_MODE_40000baseCR4_Full_BIT, 40_000_000_000},   // 24
	{"40000baseSR4_Full", unix.ETHTOOL_LINK_MODE_40000baseSR4_Full_BIT, 40_000_000_000},   // 25
	{"40000baseLR4_Full", unix.ETHTOOL_LINK_MODE_40000baseLR4_Full_BIT, 40_000_000_000},   // 26
	{"56000baseKR4_Full", unix.ETHTOOL_LINK_MODE_56000baseKR4_Full_BIT, 56_000_000_000},   // 27
	{"56000baseCR4_Full", unix.ETHTOOL_LINK_MODE_56000baseCR4_Full_BIT, 56_000_000_000},   // 28
	{"56000baseSR4_Full", unix.ETHTOOL_LINK_MODE_56000baseSR4_Full_BIT, 56_000_000_000},   // 29
	{"56000baseLR4_Full", unix.ETHTOOL_LINK_MODE_56000baseLR4_Full_BIT, 56_000_000_000},   // 30
	{"25000baseCR_Full", unix.ETHTOOL_LINK_MODE_25000baseCR_Full_BIT, 25_000_000_000},     // 31
	// Modes beyond bit 31 (require GLINKSETTINGS)
	{"25000baseKR_Full", unix.ETHTOOL_LINK_MODE_25000baseKR_Full_BIT, 25_000_000_000},                      // 32
	{"25000baseSR_Full", unix.ETHTOOL_LINK_MODE_25000baseSR_Full_BIT, 25_000_000_000},                      // 33
	{"50000baseCR2_Full", unix.ETHTOOL_LINK_MODE_50000baseCR2_Full_BIT, 50_000_000_000},                    // 34
	{"50000baseKR2_Full", unix.ETHTOOL_LINK_MODE_50000baseKR2_Full_BIT, 50_000_000_000},                    // 35
	{"100000baseKR4_Full", unix.ETHTOOL_LINK_MODE_100000baseKR4_Full_BIT, 100_000_000_000},                 // 36
	{"100000baseSR4_Full", unix.ETHTOOL_LINK_MODE_100000baseSR4_Full_BIT, 100_000_000_000},                 // 37
	{"100000baseCR4_Full", unix.ETHTOOL_LINK_MODE_100000baseCR4_Full_BIT, 100_000_000_000},                 // 38
	{"100000baseLR4_ER4_Full", unix.ETHTOOL_LINK_MODE_100000baseLR4_ER4_Full_BIT, 100_000_000_000},         // 39
	{"50000baseSR2_Full", unix.ETHTOOL_LINK_MODE_50000baseSR2_Full_BIT, 50_000_000_000},                    // 40
	{"1000baseX_Full", unix.ETHTOOL_LINK_MODE_1000baseX_Full_BIT, 1_000_000_000},                           // 41
	{"10000baseCR_Full", unix.ETHTOOL_LINK_MODE_10000baseCR_Full_BIT, 10_000_000_000},                      // 42
	{"10000baseSR_Full", unix.ETHTOOL_LINK_MODE_10000baseSR_Full_BIT, 10_000_000_000},                      // 43
	{"10000baseLR_Full", unix.ETHTOOL_LINK_MODE_10000baseLR_Full_BIT, 10_000_000_000},                      // 44
	{"10000baseLRM_Full", unix.ETHTOOL_LINK_MODE_10000baseLRM_Full_BIT, 10_000_000_000},                    // 45
	{"10000baseER_Full", unix.ETHTOOL_LINK_MODE_10000baseER_Full_BIT, 10_000_000_000},                      // 46
	{"2500baseT_Full", unix.ETHTOOL_LINK_MODE_2500baseT_Full_BIT, 2_500_000_000},                           // 47 (already present but reconfirmed)
	{"5000baseT_Full", unix.ETHTOOL_LINK_MODE_5000baseT_Full_BIT, 5_000_000_000},                           // 48
	{"FEC_NONE", unix.ETHTOOL_LINK_MODE_FEC_NONE_BIT, 0},                                                   // 49
	{"FEC_RS", unix.ETHTOOL_LINK_MODE_FEC_RS_BIT, 0},                                                       // 50 (Reed-Solomon FEC)
	{"FEC_BASER", unix.ETHTOOL_LINK_MODE_FEC_BASER_BIT, 0},                                                 // 51 (BaseR FEC)
	{"50000baseKR_Full", unix.ETHTOOL_LINK_MODE_50000baseKR_Full_BIT, 50_000_000_000},                      // 52
	{"50000baseSR_Full", unix.ETHTOOL_LINK_MODE_50000baseSR_Full_BIT, 50_000_000_000},                      // 53
	{"50000baseCR_Full", unix.ETHTOOL_LINK_MODE_50000baseCR_Full_BIT, 50_000_000_000},                      // 54
	{"50000baseLR_ER_FR_Full", unix.ETHTOOL_LINK_MODE_50000baseLR_ER_FR_Full_BIT, 50_000_000_000},          // 55
	{"50000baseDR_Full", unix.ETHTOOL_LINK_MODE_50000baseDR_Full_BIT, 50_000_000_000},                      // 56
	{"100000baseKR2_Full", unix.ETHTOOL_LINK_MODE_100000baseKR2_Full_BIT, 100_000_000_000},                 // 57
	{"100000baseSR2_Full", unix.ETHTOOL_LINK_MODE_100000baseSR2_Full_BIT, 100_000_000_000},                 // 58
	{"100000baseCR2_Full", unix.ETHTOOL_LINK_MODE_100000baseCR2_Full_BIT, 100_000_000_000},                 // 59
	{"100000baseLR2_ER2_FR2_Full", unix.ETHTOOL_LINK_MODE_100000baseLR2_ER2_FR2_Full_BIT, 100_000_000_000}, // 60
	{"100000baseDR2_Full", unix.ETHTOOL_LINK_MODE_100000baseDR2_Full_BIT, 100_000_000_000},                 // 61
	{"200000baseKR4_Full", unix.ETHTOOL_LINK_MODE_200000baseKR4_Full_BIT, 200_000_000_000},                 // 62
	{"200000baseSR4_Full", unix.ETHTOOL_LINK_MODE_200000baseSR4_Full_BIT, 200_000_000_000},                 // 63
	{"200000baseLR4_ER4_FR4_Full", unix.ETHTOOL_LINK_MODE_200000baseLR4_ER4_FR4_Full_BIT, 200_000_000_000}, // 64
	{"200000baseDR4_Full", unix.ETHTOOL_LINK_MODE_200000baseDR4_Full_BIT, 200_000_000_000},                 // 65
	{"200000baseCR4_Full", unix.ETHTOOL_LINK_MODE_200000baseCR4_Full_BIT, 200_000_000_000},                 // 66
	{"100baseT1_Full", unix.ETHTOOL_LINK_MODE_100baseT1_Full_BIT, 100_000_000},                             // 67 (Automotive/SPE)
	{"1000baseT1_Full", unix.ETHTOOL_LINK_MODE_1000baseT1_Full_BIT, 1_000_000_000},                         // 68 (Automotive/SPE)
	{"400000baseKR8_Full", unix.ETHTOOL_LINK_MODE_400000baseKR8_Full_BIT, 400_000_000_000},                 // 69
	{"400000baseSR8_Full", unix.ETHTOOL_LINK_MODE_400000baseSR8_Full_BIT, 400_000_000_000},                 // 70
	{"400000baseLR8_ER8_FR8_Full", unix.ETHTOOL_LINK_MODE_400000baseLR8_ER8_FR8_Full_BIT, 400_000_000_000}, // 71
	{"400000baseDR8_Full", unix.ETHTOOL_LINK_MODE_400000baseDR8_Full_BIT, 400_000_000_000},                 // 72
	{"400000baseCR8_Full", unix.ETHTOOL_LINK_MODE_400000baseCR8_Full_BIT, 400_000_000_000},                 // 73
	{"FEC_LLRS", unix.ETHTOOL_LINK_MODE_FEC_LLRS_BIT, 0},                                                   // 74 (Low Latency Reed-Solomon FEC)
	// PAM4 modes start here? Often indicated by lack of KR/CR/SR/LR or different naming
	{"100000baseKR_Full", unix.ETHTOOL_LINK_MODE_100000baseKR_Full_BIT, 100_000_000_000},                   // 75 (Likely 100GBASE-KR1)
	{"100000baseSR_Full", unix.ETHTOOL_LINK_MODE_100000baseSR_Full_BIT, 100_000_000_000},                   // 76 (Likely 100GBASE-SR1)
	{"100000baseLR_ER_FR_Full", unix.ETHTOOL_LINK_MODE_100000baseLR_ER_FR_Full_BIT, 100_000_000_000},       // 77 (Likely 100GBASE-LR1/ER1/FR1)
	{"100000baseCR_Full", unix.ETHTOOL_LINK_MODE_100000baseCR_Full_BIT, 100_000_000_000},                   // 78 (Likely 100GBASE-CR1)
	{"100000baseDR_Full", unix.ETHTOOL_LINK_MODE_100000baseDR_Full_BIT, 100_000_000_000},                   // 79
	{"200000baseKR2_Full", unix.ETHTOOL_LINK_MODE_200000baseKR2_Full_BIT, 200_000_000_000},                 // 80 (Likely 200GBASE-KR2)
	{"200000baseSR2_Full", unix.ETHTOOL_LINK_MODE_200000baseSR2_Full_BIT, 200_000_000_000},                 // 81 (Likely 200GBASE-SR2)
	{"200000baseLR2_ER2_FR2_Full", unix.ETHTOOL_LINK_MODE_200000baseLR2_ER2_FR2_Full_BIT, 200_000_000_000}, // 82 (Likely 200GBASE-LR2/etc)
	{"200000baseDR2_Full", unix.ETHTOOL_LINK_MODE_200000baseDR2_Full_BIT, 200_000_000_000},                 // 83
	{"200000baseCR2_Full", unix.ETHTOOL_LINK_MODE_200000baseCR2_Full_BIT, 200_000_000_000},                 // 84 (Likely 200GBASE-CR2)
	{"400000baseKR4_Full", unix.ETHTOOL_LINK_MODE_400000baseKR4_Full_BIT, 400_000_000_000},                 // 85 (Likely 400GBASE-KR4)
	{"400000baseSR4_Full", unix.ETHTOOL_LINK_MODE_400000baseSR4_Full_BIT, 400_000_000_000},                 // 86 (Likely 400GBASE-SR4)
	{"400000baseLR4_ER4_FR4_Full", unix.ETHTOOL_LINK_MODE_400000baseLR4_ER4_FR4_Full_BIT, 400_000_000_000}, // 87 (Likely 400GBASE-LR4/etc)
	{"400000baseDR4_Full", unix.ETHTOOL_LINK_MODE_400000baseDR4_Full_BIT, 400_000_000_000},                 // 88
	{"400000baseCR4_Full", unix.ETHTOOL_LINK_MODE_400000baseCR4_Full_BIT, 400_000_000_000},                 // 89 (Likely 400GBASE-CR4)
	{"100baseFX_Half", unix.ETHTOOL_LINK_MODE_100baseFX_Half_BIT, 100_000_000},                             // 90
	{"100baseFX_Full", unix.ETHTOOL_LINK_MODE_100baseFX_Full_BIT, 100_000_000},                             // 91
}
