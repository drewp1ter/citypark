package address

import "testing"

func TestRecipientFields(t *testing.T) {
	in := CleanFields(Fields{RecipientName: " Иван ", RecipientPhone: "8 (999) 123-45-67", City: "Город", Street: "Дом"})
	if in.RecipientName != "Иван" || in.RecipientPhone != "+79991234567" {
		t.Fatal(in)
	}
	if errors := FieldErrors(in); len(errors) != 0 {
		t.Fatal(errors)
	}
	in.RecipientName = "я"
	in.RecipientPhone = "123"
	errors := FieldErrors(in)
	if errors["recipientName"] != "Укажите имя" || errors["recipientPhone"] == "" {
		t.Fatal(errors)
	}
}

func TestRecipientPersistence(t *testing.T) {
	_, service := bookFixture(t)
	entry := createBookEntry(t, service, 1, Fields{RecipientName: " Иван ", RecipientPhone: "8 (999) 123-45-67", City: "Город", Street: "Дом"})
	if entry.RecipientName != "Иван" || entry.RecipientPhone != "+79991234567" {
		t.Fatal(entry)
	}
	entry, err := service.UpdateEntry(t.Context(), 1, entry.ID, Fields{RecipientName: "Мария", RecipientPhone: "+79991234568", City: "Город", Street: "Дом"})
	if err != nil || entry.RecipientName != "Мария" || entry.RecipientPhone != "+79991234568" {
		t.Fatal(entry, err)
	}
}
