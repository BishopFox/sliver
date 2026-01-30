// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/go-teams-notify
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package adaptivecard

// supportedElementTypes returns a list of valid types for an Adaptive Card
// element used in Microsoft Teams messages. This list is intended to be used
// for validation and display purposes.
func supportedElementTypes() []string {
	// TODO: Confirm whether all types are supported.
	//
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference#support-for-adaptive-cards
	// https://adaptivecards.io/explorer/AdaptiveCard.html
	return []string{
		TypeElementActionSet,
		TypeElementColumnSet,
		TypeElementContainer,
		TypeElementFactSet,
		TypeElementImage,
		TypeElementImageSet,
		TypeElementInputChoiceSet,
		TypeElementInputDate,
		TypeElementInputNumber,
		TypeElementInputText,
		TypeElementInputTime,
		TypeElementInputToggle,
		TypeElementMedia, // Introduced in version 1.1 (TODO: Is this supported in Teams message?)
		TypeElementRichTextBlock,
		TypeElementTable, // Introduced in version 1.5
		TypeElementTextBlock,
		TypeElementTextRun,
		TypeElementMSTeamsCodeBlock,
	}
}

// supportedSizeValues returns a list of valid Size values for applicable
// Element types. This list is intended to be used for validation and display
// purposes.
func supportedSizeValues() []string {
	// https://adaptivecards.io/explorer/TextBlock.html
	return []string{
		SizeSmall,
		SizeDefault,
		SizeMedium,
		SizeLarge,
		SizeExtraLarge,
	}
}

// supportedWeightValues returns a list of valid Weight values for text in
// applicable Element types. This list is intended to be used for validation
// and display purposes.
func supportedWeightValues() []string {
	// https://adaptivecards.io/explorer/TextBlock.html
	return []string{
		WeightBolder,
		WeightLighter,
		WeightDefault,
	}
}

// supportedColorValues returns a list of valid Color values for text in
// applicable Element types. This list is intended to be used for validation
// and display purposes.
func supportedColorValues() []string {
	// https://adaptivecards.io/explorer/TextBlock.html
	return []string{
		ColorDefault,
		ColorDark,
		ColorLight,
		ColorAccent,
		ColorGood,
		ColorWarning,
		ColorAttention,
	}
}

// supportedSpacingValues returns a list of valid Spacing values for Element
// types. This list is intended to be used for validation and display
// purposes.
func supportedSpacingValues() []string {
	// https://adaptivecards.io/explorer/TextBlock.html
	return []string{
		SpacingDefault,
		SpacingNone,
		SpacingSmall,
		SpacingMedium,
		SpacingLarge,
		SpacingExtraLarge,
		SpacingPadding,
	}
}

// supportedHorizontalAlignmentValues returns a list of valid horizontal
// alignment values for supported container and text types. This list is
// intended to be used for validation and display purposes.
func supportedHorizontalAlignmentValues() []string {
	// https://adaptivecards.io/explorer/Table.html
	// https://adaptivecards.io/explorer/TextBlock.html
	// https://adaptivecards.io/schemas/adaptive-card.json
	return []string{
		HorizontalAlignmentLeft,
		HorizontalAlignmentCenter,
		HorizontalAlignmentRight,
	}
}

// supportedVerticalAlignmentValues returns a list of valid vertical content
// alignment values for supported container types. This list is intended to be
// used for validation and display purposes.
func supportedVerticalContentAlignmentValues() []string {
	// https://adaptivecards.io/explorer/Table.html
	// https://adaptivecards.io/schemas/adaptive-card.json
	return []string{
		VerticalAlignmentTop,
		VerticalAlignmentCenter,
		VerticalAlignmentBottom,
	}
}

// supportedActionValues accepts a value indicating the maximum Adaptive Card
// schema version supported and returns a list of valid Action types. This
// list is intended to be used for validation and display purposes.
//
// NOTE: See also the supportedISelectActionValues() function. See ref links
// for unsupported Action types.
func supportedActionValues(version float64) []string {
	// https://adaptivecards.io/explorer/AdaptiveCard.html
	// https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/universal-action-model
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference
	supportedValues := []string{
		TypeActionOpenURL,
		TypeActionShowCard,
		TypeActionToggleVisibility,

		// Action.Submit is not supported for Adaptive Cards in Incoming
		// Webhooks.
		//
		// TypeActionSubmit,
	}

	// Version 1.4 is when Action.Execute was introduced.
	//
	// Per this doc:
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference
	//
	// the "Action.Execute" action is supported:
	//
	// "For Adaptive Cards in Incoming Webhooks, all native Adaptive Card
	// schema elements, except Action.Submit, are fully supported. The
	// supported actions are Action.OpenURL, Action.ShowCard,
	// Action.ToggleVisibility, and Action.Execute."
	if version >= ActionExecuteMinCardVersionRequired {
		supportedValues = append(supportedValues, TypeActionExecute)
	}

	return supportedValues
}

// supportedISelectActionValues accepts a value indicating the maximum
// Adaptive Card schema version supported and returns a list of valid
// ISelectAction types. This list is intended to be used for validation and
// display purposes.
//
// NOTE: See also the supportedActionValues() function. See ref links for
// unsupported Action types.
func supportedISelectActionValues(version float64) []string {
	// https://adaptivecards.io/explorer/Column.html
	// https://adaptivecards.io/explorer/TableCell.html
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference
	supportedValues := []string{
		TypeActionOpenURL,
		TypeActionToggleVisibility,

		// Action.Submit is not supported for Adaptive Cards in Incoming
		// Webhooks.
		//
		// TypeActionSubmit,

		// Action.ShowCard is not a supported Action for selectAction fields
		// (ISelectAction).
		//
		// TypeActionShowCard,
	}

	// Version 1.4 is when Action.Execute was introduced.
	//
	// Per this doc:
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference
	//
	// the "Action.Execute" action is supported:
	//
	// "For Adaptive Cards in Incoming Webhooks, all native Adaptive Card
	// schema elements, except Action.Submit, are fully supported. The
	// supported actions are Action.OpenURL, Action.ShowCard,
	// Action.ToggleVisibility, and Action.Execute."
	if version >= ActionExecuteMinCardVersionRequired {
		supportedValues = append(supportedValues, TypeActionExecute)
	}

	return supportedValues
}

// supportedAttachmentLayoutValues returns a list of valid AttachmentLayout
// values for Message type. This list is intended to be used for validation
// and display purposes.
//
// NOTE: See also the supportedActionValues() function.
func supportedAttachmentLayoutValues() []string {
	return []string{
		AttachmentLayoutList,
		AttachmentLayoutCarousel,
	}
}

// supportedStyleValues returns a list of valid Style field values for the
// specified element type. This list is intended to be used for validation and
// display purposes.
func supportedStyleValues(elementType string) []string {
	switch elementType {
	case TypeElementColumnSet:
		return supportedContainerStyleValues()
	case TypeElementContainer:
		return supportedContainerStyleValues()
	case TypeElementTable:
		return supportedContainerStyleValues()
	case TypeElementImage:
		return supportedImageStyleValues()
	case TypeElementInputChoiceSet:
		return supportedChoiceInputStyleValues()
	case TypeElementInputText:
		return supportedTextInputStyleValues()
	case TypeElementTextBlock:
		return supportedTextBlockStyleValues()

	// Unsupported element types are indicated by an explicit empty list.
	default:
		return []string{}
	}
}

// supportedImageStyleValues returns a list of valid Style field values for
// the Image element type. This list is intended to be used for validation and
// display purposes.
func supportedImageStyleValues() []string {
	return []string{
		ImageStyleDefault,
		ImageStylePerson,
	}
}

// supportedChoiceInputStyleValues returns a list of valid Style field values
// for ChoiceInput related element types (e.g., Input.ChoiceSet) This list is
// intended to be used for validation and display purposes.
func supportedChoiceInputStyleValues() []string {
	return []string{
		ChoiceInputStyleCompact,
		ChoiceInputStyleExpanded,
		ChoiceInputStyleFiltered,
	}
}

// supportedTextInputStyleValues returns a list of valid Style field values
// for TextInput related element types (e.g., Input.Text) This list is
// intended to be used for validation and display purposes.
func supportedTextInputStyleValues() []string {
	return []string{
		TextInputStyleText,
		TextInputStyleTel,
		TextInputStyleURL,
		TextInputStyleEmail,
		TextInputStylePassword,
	}
}

// supportedTextBlockStyleValues returns a list of valid Style field values
// for the TextBlock element type. This list is intended to be used for
// validation and display purposes.
func supportedTextBlockStyleValues() []string {
	return []string{
		TextBlockStyleDefault,
		TextBlockStyleHeading,
	}
}

// supportedContainerStyleValues returns a list of valid Style field values
// for Container types (e.g., Column, ColumnSet, Container). This list is
// intended to be used for validation and display purposes.
func supportedContainerStyleValues() []string {
	return []string{
		ContainerStyleDefault,
		ContainerStyleEmphasis,
		ContainerStyleGood,
		ContainerStyleAttention,
		ContainerStyleWarning,
		ContainerStyleAccent,
	}
}

// supportedMSTeamsWidthValues returns a list of valid Width field values for
// MSTeams type. This list is intended to be used for validation and display
// purposes.
func supportedMSTeamsWidthValues() []string {
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format#full-width-adaptive-card
	return []string{
		MSTeamsWidthFull,
	}
}

// supportedActionFallbackValues accepts a value indicating the maximum
// Adaptive Card schema version supported and returns a list of valid Action
// Fallback types. This list is intended to be used for validation and display
// purposes.
func supportedActionFallbackValues(version float64) []string {
	// https://adaptivecards.io/explorer/Action.OpenUrl.html
	// https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/universal-action-model
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference
	supportedValues := supportedActionValues(version)
	supportedValues = append(supportedValues, TypeFallbackOptionDrop)

	return supportedValues
}

// supportedISelectActionFallbackValues accepts a value indicating the maximum
// Adaptive Card schema version supported and returns a list of valid
// ISelectAction Fallback types. This list is intended to be used for
// validation and display purposes.
func supportedISelectActionFallbackValues(version float64) []string {
	// https://adaptivecards.io/explorer/Action.OpenUrl.html
	// https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/universal-action-model
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference
	supportedValues := supportedISelectActionValues(version)
	supportedValues = append(supportedValues, TypeFallbackOptionDrop)

	return supportedValues
}
