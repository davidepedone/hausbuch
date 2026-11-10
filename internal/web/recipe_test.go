package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// recipe creates a recipe with the ingredients as its "Ingredients" list and returns its check page's path.
func (b *browser) recipe(ingredients ...string) string {
	b.app.t.Helper()
	body := "## Ingredients\n\n- " + strings.Join(ingredients, "\n- ") + "\n\n## Steps\n\n- Mix everything\n"
	resp := b.post("/pages", url.Values{"title": {"Pancakes"}, "type": {"recipe"}, "body": {body}})
	if resp.StatusCode != http.StatusSeeOther {
		b.app.t.Fatalf("create recipe: %s\n%s", resp.Status, resp.body)
	}
	return resp.Header.Get("Location") + "/pantry"
}

// checked returns the check page's status line of each ingredient, in order.
func checked(t *testing.T, body string) []string {
	t.Helper()
	var lines []string
	for _, part := range strings.Split(body, `<p class="mt-1 break-words text-sm `)[1:] {
		_, line, _ := strings.Cut(part, ">")
		line, _, _ = strings.Cut(line, "</p>")
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func TestCheckTheRecipe(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Flour"}, "location": {"Cupboard"}, "quantity": {"100"}, "unit": {"g"}})
	b.addEntry(url.Values{"name": {"Butter"}})
	path := b.recipe("200 g flour", "Eggs", "50 g butter")
	mustContain(t, b.view(strings.TrimSuffix(path, "/pantry")), `href="`+path+`"`, "Check the pantry")

	body := b.view(path)
	want := []string{`In the pantry: Flour in <a href="/locations/Cupboard" class="underline decoration-stone-300 hover:text-stone-700">Cupboard</a> · 100 g`, "Missing", "On the shopping list"}
	if got := checked(t, body); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("states %q, want %q", got, want)
	}
	if strings.Contains(body, "Mix everything") {
		t.Error("a step is listed as an ingredient")
	}
	mustContain(t, body,
		`<input type="checkbox" name="add" value="0" aria-label="Add 200 g flour"`+"\n",
		`name="add" value="1" aria-label="Add Eggs" checked`,
		`<input type="checkbox" name="add" value="2" aria-label="Add 50 g butter"`+"\n",
		`name="name-0" value="200 g flour"`)
}

func TestAgeOfAMatchedRow(t *testing.T) {
	a := newApp(t)
	now := a.stockClock()
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Flour"}})
	path := b.recipe("200 g flour")
	for _, c := range []struct {
		later time.Time
		want  string
	}{
		{now.AddDate(0, 2, 0), "In the pantry: Flour · 1"},
		{now.AddDate(0, 3, -1), "In the pantry: Flour · 1"},
		{now.AddDate(0, 8, 0), `In the pantry: Flour · 1 · <span class="font-medium text-amber-700">changed 8 months ago</span>`},
	} {
		*now = c.later
		if got := checked(t, b.view(path)); len(got) != 1 || got[0] != c.want {
			t.Errorf("on %s: %q, want %q", c.later.Format(time.DateOnly), got, c.want)
		}
	}
}

func TestRecipeWithoutIngredients(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	for _, body := range []string{"## Steps\n\n- Mix\n", "## Ingredients\n\nWhatever is left.\n"} {
		resp := b.post("/pages", url.Values{"title": {"Leftovers " + body[:8]}, "type": {"recipe"}, "body": {body}})
		mustContain(t, b.view(resp.Header.Get("Location")+"/pantry"), "This recipe lists no ingredients")
	}
}

func TestOnlyRecipesAreChecked(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	resp := b.post("/pages", url.Values{"title": {"Descaling"}, "type": {"howto"}, "body": {"## Ingredients\n\n- Citric acid\n"}})
	path := resp.Header.Get("Location")
	if strings.Contains(b.view(path), "Check the pantry") {
		t.Error("check offered on a how-to")
	}
	for _, r := range []response{b.get(path + "/pantry"), b.post(path+"/pantry", url.Values{"add": {"0"}, "name-0": {"Citric acid"}})} {
		if r.StatusCode != http.StatusNotFound {
			t.Errorf("how-to check: %s", r.Status)
		}
	}
	a.noEntries()
}

func TestAddTheMissingOnes(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Flour"}})
	path := b.recipe("200 g flour", "Eggs")
	// The form as shown: only the missing one ticked.
	resp := b.post(path, url.Values{"add": {"1"}, "name-0": {"200 g flour"}, "name-1": {"Eggs"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/shopping" {
		t.Fatalf("add: %s %s", resp.Status, resp.Header.Get("Location"))
	}
	if e := a.entry("Eggs"); e.Section != stock.Pantry || e.Quantity != 0 {
		t.Errorf("entry %+v", e)
	}
	list, _ := a.stock.Entries(t.Context())
	if len(list) != 1 {
		t.Errorf("list %+v", list)
	}
}

func TestRenameBeforeAdding(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.recipe("3 eggs", "Salt")
	b.post(path, url.Values{"add": {"0"}, "name-0": {"Eggs"}, "name-1": {"Salt"}})
	a.entry("Eggs")
	list, _ := a.stock.Entries(t.Context())
	if len(list) != 1 {
		t.Errorf("unticked salt added too: %+v", list)
	}
}

func TestNothingTicked(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.recipe("3 eggs", "Salt")
	resp := b.post(path, url.Values{"name-0": {"Eggs"}, "name-1": {"Salt"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("nothing ticked: %s", resp.Status)
	}
	mustContain(t, resp.body, "Tick what to add.", `name="name-0" value="Eggs"`,
		`<input type="checkbox" name="add" value="0" aria-label="Add 3 eggs"`+"\n")
	a.noEntries()
}

func TestIngredientMergesWithTheList(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addEntry(url.Values{"name": {"Milk"}, "section": {"pantry"}, "quantity": {"1"}})
	path := b.recipe("Eggs")
	b.post(path, url.Values{"add": {"0"}, "name-0": {"milk"}})
	if list, _ := a.stock.Entries(t.Context()); len(list) != 1 || list[0].Name != "Milk" {
		t.Errorf("not merged: %+v", list)
	}
}
