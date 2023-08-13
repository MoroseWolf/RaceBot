package main

type Kb struct {
	Inline  bool       `json:"inline,omitempty"`
	Buttons [][]Button `json:"buttons"`
}

type Button struct {
	Action ActionBtn `json:"action"`
	Color  string    `json:"color,omitempty"`
}

type ActionBtn struct {
	TypeAction string `json:"type"`
	Link       string `json:"link,omitempty"`
	Label      string `json:"label,omitempty"`
	Payload    string `json:"payload,omitempty"`
}

type Carousel struct {
	Type     string         `json:"type"`
	Elements []CarouselItem `json:"elements"`
}

type CarouselItem struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	PhotoID     string    `json:"photo_id,omitempty"`
	Action      ActionBtn `json:"action"`
	Buttons     []Button  `json:"buttons"`
}

type Playload struct {
	Command string `json:"command"`
}
