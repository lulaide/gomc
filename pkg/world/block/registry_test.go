package block

import "testing"

type torchDefinition struct{}

func (t *torchDefinition) ID() int {
	return 75
}

func (t *torchDefinition) IsAssociatedBlockID(otherID int) bool {
	return otherID == 75 || otherID == 76
}

type propertyDefinition struct {
	id       int
	opacity  int
	movement bool
	liquid   bool
	tile     bool
}

func (d *propertyDefinition) ID() int {
	return d.id
}

func (d *propertyDefinition) IsAssociatedBlockID(otherID int) bool {
	return d.id == otherID
}

func (d *propertyDefinition) GetLightOpacity() int {
	return d.opacity
}

func (d *propertyDefinition) BlocksMovement() bool {
	return d.movement
}

func (d *propertyDefinition) IsLiquid() bool {
	return d.liquid
}

func (d *propertyDefinition) IsTileEntityProvider() bool {
	return d.tile
}

func TestIsAssociatedBlockID(t *testing.T) {
	ResetRegistry()
	Register(NewBaseDefinition(1))
	Register(&torchDefinition{})
	Register(NewBaseDefinition(76))

	if !IsAssociatedBlockID(1, 1) {
		t.Fatal("same id should be associated")
	}
	if IsAssociatedBlockID(1, 2) {
		t.Fatal("unregistered second id should not be associated")
	}
	if !IsAssociatedBlockID(75, 76) {
		t.Fatal("torch ids should be associated by override")
	}
	if IsAssociatedBlockID(0, 76) {
		t.Fatal("id 0 should not be associated unless identical")
	}
}

func TestBlockPropertyLookups(t *testing.T) {
	ResetRegistry()
	Register(&propertyDefinition{
		id:       7,
		opacity:  255,
		movement: true,
		liquid:   false,
		tile:     true,
	})

	if got := GetLightOpacity(7); got != 255 {
		t.Fatalf("light opacity mismatch: got=%d want=255", got)
	}
	if !BlocksMovement(7) {
		t.Fatal("expected movement-blocking material")
	}
	if IsLiquid(7) {
		t.Fatal("expected non-liquid material")
	}
	if !IsTileEntityProvider(7) {
		t.Fatal("expected tile-entity-provider flag")
	}

	SetLightOpacity(7, 3)
	SetMaterialProperties(7, false, true)
	SetTileEntityProvider(7, false)

	if got := GetLightOpacity(7); got != 3 {
		t.Fatalf("light opacity override mismatch: got=%d want=3", got)
	}
	if BlocksMovement(7) {
		t.Fatal("movement flag override failed")
	}
	if !IsLiquid(7) {
		t.Fatal("liquid flag override failed")
	}
	if IsTileEntityProvider(7) {
		t.Fatal("tile provider override failed")
	}
}
