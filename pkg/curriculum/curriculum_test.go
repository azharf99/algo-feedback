package curriculum

import (
	"strings"
	"testing"
)

func TestGetCourseLevel(t *testing.T) {
	tests := []struct {
		module   string
		expected string
	}{
		{"Artificial Intelligence, AI ENG", "Artificial"},
		{"Python Start 1st year IND", "Python"},
		{"Visual programming INDONESIA", "Visual"},
		{"Frontend Development_ENG", "Frontend"},
		{"Unknown Module", "Unknown"},
		{"SingleWord", "SingleWord"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			got := GetCourseLevel(tt.module)
			if got != tt.expected {
				t.Errorf("GetCourseLevel(%q) = %q; want %q", tt.module, got, tt.expected)
			}
		})
	}
}

func TestGetTopic(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		number   int
		expected string
	}{
		{
			"Valid AI month 1",
			"Artificial Intelligence, AI ENG",
			1,
			"Introduction to Artificial Intelligence (AI) and Creating Interactive Stories with AI",
		},
		{
			"Valid AI month 2",
			"Artificial Intelligence, AI ENG",
			2,
			"Introduction to Programming, AI, and Interactive Digital Creation",
		},
		{
			"Unknown Topic",
			"Unknown Topic",
			1,
			"",
		},
		{
			"Unknown Month for existing Topic",
			"Artificial Intelligence, AI ENG",
			99,
			"", // Assuming month 99 is not defined or is empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTopic(tt.topic, tt.number)
			if got != tt.expected {
				t.Errorf("GetTopic(%q, %d) = %q; want %q", tt.topic, tt.number, got, tt.expected)
			}
		})
	}
}

func TestGetResult(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		number   int
		expected string
	}{
		{
			"Valid AI month 2",
			"Artificial Intelligence, AI ENG",
			2,
			"In this module, students learn the basics of programming languages, how web pages work, and the differences between HTML, CSS, and programming languages. They also create games in Twine, customize their appearance with the help of ChatGPT, and understand the importance of editing code independently. In addition, students are introduced to AI concepts such as graph neural networks, diffusion models, and image generation tools like DALL·E 3 and FusionBrain. They learn to craft prompts effectively to generate images and provide constructive feedback on their peers' projects",
		},
		{
			"Unknown Topic",
			"Unknown Topic",
			1,
			"",
		},
		{
			"Unknown Month for existing Topic",
			"Artificial Intelligence, AI ENG",
			99,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetResult(tt.topic, tt.number)
			if got != tt.expected {
				t.Errorf("GetResult(%q, %d) = %q; want %q", tt.topic, tt.number, got, tt.expected)
			}
		})
	}
}

func TestGetCompetency(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		number   int
		expected string
	}{
		{
			"Valid AI month 1",
			"Artificial Intelligence, AI ENG",
			1,
			"In this module, students were introduced to the fundamentals of Artificial Intelligence (AI), large language models like ChatGPT, and how to craft effective prompts. They learned to utilize this technology to create storylines, develop ideas for interactive games using Twine, and use Miro to document and organize their ideas.Through this process, students not only sharpened their creativity and writing skills but also learned to think systematically, solve problems, and use digital tools productively for everyday tasks and learning",
		},
		{
			"Unknown Topic",
			"Unknown Topic",
			1,
			"",
		},
		{
			"Unknown Month for existing Topic",
			"Artificial Intelligence, AI ENG",
			99,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCompetency(tt.topic, tt.number)
			if got != tt.expected {
				t.Errorf("GetCompetency(%q, %d) = %q; want %q", tt.topic, tt.number, got, tt.expected)
			}
		})
	}
}

func TestGetTutorFeedback(t *testing.T) {
	tests := []struct {
		name        string
		studentName string
	}{
		{"Normal Name", "Budi"},
		{"Long Name", "A Very Long Student Name That Should Still Work"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTutorFeedback(tt.studentName)

			// Count occurrences of studentName in the feedback
			count := strings.Count(got, tt.studentName)

			// According to the format string, there are 6 %s placeholders for the student name
			expectedOccurrences := 6

			if count != expectedOccurrences {
				t.Errorf("GetTutorFeedback() expected %d occurrences of %q, got %d. Output: %q", expectedOccurrences, tt.studentName, count, got)
			}

			if !strings.Contains(got, "Azhar Faturohman Ahidin") {
				t.Errorf("GetTutorFeedback() missing tutor name")
			}
		})
	}

    // Special case for empty string since strings.Count handles empty strings differently
    t.Run("Empty Name", func(t *testing.T) {
        got := GetTutorFeedback("")
        if !strings.Contains(got, "Halo, Ayah/Bunda dari ! 👋") {
            t.Errorf("GetTutorFeedback() with empty name did not format as expected. Output: %q", got)
        }
    })
}
