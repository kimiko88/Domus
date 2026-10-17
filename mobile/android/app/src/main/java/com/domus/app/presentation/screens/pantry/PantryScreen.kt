package com.domus.app.presentation.screens.pantry

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.*
import com.domus.app.presentation.theme.*
import com.domus.app.presentation.viewmodel.DashboardViewModel
import java.time.LocalDate

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PantryScreen(
    viewModel: DashboardViewModel,
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    var selectedTab by remember { mutableStateOf(0) }
    val tabTitles = listOf("Dispensa", "Ricettario", "Calendario Pasti", "Scanner Scontrino")

    LaunchedEffect(Unit) {
        viewModel.loadPantry()
        viewModel.loadRecipes()
        viewModel.loadMealPlans()
    }

    val pantryItems by viewModel.pantryItems.collectAsState()
    val recipes by viewModel.recipes.collectAsState()
    val mealPlans by viewModel.mealPlans.collectAsState()
    val isLoading by viewModel.isPantryLoading.collectAsState()

    var showAddItemDialog by remember { mutableStateOf(false) }
    var showAddRecipeDialog by remember { mutableStateOf(false) }
    var showAddMealDialog by remember { mutableStateOf(false) }
    var selectedRecipeForMeal by remember { mutableStateOf<RecipeDto?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(Slate900)
    ) {
        // Top App Bar
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            IconButton(
                onClick = onDismiss,
                modifier = Modifier.sizeIn(minWidth = 48.dp, minHeight = 48.dp)
            ) {
                Text("✕", fontSize = 22.sp, color = Color.White)
            }

            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = "Dispensa & Ricettario",
                    fontWeight = FontWeight.Bold,
                    fontSize = 18.sp,
                    color = Color.White
                )
                Text(
                    text = "Gestione scorte alimentari e pasti smart",
                    fontSize = 12.sp,
                    color = Color.White.copy(alpha = 0.7f)
                )
            }

            // Quick Refresh Button
            IconButton(
                onClick = {
                    viewModel.loadPantry()
                    viewModel.loadRecipes()
                    viewModel.loadMealPlans()
                },
                modifier = Modifier.sizeIn(minWidth = 48.dp, minHeight = 48.dp)
            ) {
                Text("🔄", fontSize = 18.sp)
            }
        }

        // Navigation Tabs
        SecondaryTabRow(
            selectedTabIndex = selectedTab,
            containerColor = Slate800,
            contentColor = CyanAccent,
            modifier = Modifier.fillMaxWidth()
        ) {
            tabTitles.forEachIndexed { index, title ->
                Tab(
                    selected = selectedTab == index,
                    onClick = { selectedTab = index },
                    modifier = Modifier.heightIn(min = 48.dp),
                    text = {
                        Text(
                            text = title,
                            fontWeight = if (selectedTab == index) FontWeight.Bold else FontWeight.Normal,
                            fontSize = 13.sp,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                )
            }
        }

        // Status Banner for Alerts
        AnimatedVisibility(visible = statusMessage != null, enter = fadeIn(), exit = fadeOut()) {
            statusMessage?.let { msg ->
                Surface(
                    color = EmeraldSuccess.copy(alpha = 0.2f),
                    border = BorderStroke(1.dp, EmeraldSuccess.copy(alpha = 0.5f)),
                    shape = RoundedCornerShape(8.dp),
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 6.dp)
                ) {
                    Row(
                        modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text(msg, color = EmeraldSuccess, fontSize = 13.sp, fontWeight = FontWeight.SemiBold)
                        IconButton(onClick = { statusMessage = null }, modifier = Modifier.size(24.dp)) {
                            Text("✕", color = EmeraldSuccess, fontSize = 12.sp)
                        }
                    }
                }
            }
        }

        // Tab Content
        Box(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth()
        ) {
            when (selectedTab) {
                0 -> PantryInventoryTabContent(
                    items = pantryItems,
                    isLoading = isLoading,
                    onUpdateStock = { id, delta -> viewModel.updatePantryStock(id, delta) },
                    onDelete = { id -> viewModel.deletePantryItem(id) },
                    onAddToShopping = { name, category ->
                        viewModel.createShopping(name = name, category = category, quantity = "1")
                        statusMessage = "Aggiunto alla lista della spesa: $name"
                    },
                    onOpenAddDialog = { showAddItemDialog = true }
                )
                1 -> PantryRecipesTabContent(
                    recipes = recipes,
                    isLoading = isLoading,
                    onPlanMeal = { recipe ->
                        selectedRecipeForMeal = recipe
                        showAddMealDialog = true
                    },
                    onDelete = { id -> viewModel.deleteRecipe(id) },
                    onOpenAddDialog = { showAddRecipeDialog = true }
                )
                2 -> PantryMealCalendarTabContent(
                    mealPlans = mealPlans,
                    recipes = recipes,
                    isLoading = isLoading,
                    onConsumeMeal = { planId, persons ->
                        viewModel.consumeMealPlan(planId, persons) { res ->
                            if (res != null) {
                                val alerts = if (res.low_stock_alerts.isNotEmpty()) {
                                    " (Attenzione sotto-soglia: ${res.low_stock_alerts.joinToString(", ")})"
                                } else ""
                                statusMessage = "Pasto consumato! Scorte dispensa decurtate.$alerts"
                            }
                        }
                    },
                    onDeleteMeal = { id -> viewModel.deleteMealPlan(id) },
                    onOpenAddDialog = {
                        selectedRecipeForMeal = null
                        showAddMealDialog = true
                    }
                )
                3 -> PantryReceiptScannerTabContent(
                    viewModel = viewModel,
                    onImportSuccess = { count, total ->
                        statusMessage = "Scontrino importato! $count prodotti aggiornati, €%.2f spesa registrata.".format(total)
                        selectedTab = 0
                    }
                )
            }
        }
    }

    // Modal Dialogs
    if (showAddItemDialog) {
        AddPantryItemDialog(
            onDismiss = { showAddItemDialog = false },
            onConfirm = { req ->
                viewModel.createPantryItem(req) { success ->
                    if (success) {
                        showAddItemDialog = false
                        statusMessage = "Alimento ${req.name} aggiunto in dispensa!"
                    }
                }
            }
        )
    }

    if (showAddRecipeDialog) {
        AddRecipeDialog(
            pantryItems = pantryItems,
            onDismiss = { showAddRecipeDialog = false },
            onConfirm = { req ->
                viewModel.createRecipe(req) { success ->
                    if (success) {
                        showAddRecipeDialog = false
                        statusMessage = "Ricetta ${req.title} salvata nel ricettario!"
                    }
                }
            }
        )
    }

    if (showAddMealDialog) {
        AddMealPlanDialog(
            preselectedRecipe = selectedRecipeForMeal,
            recipes = recipes,
            onDismiss = { showAddMealDialog = false },
            onConfirm = { req ->
                viewModel.createMealPlan(req) { success ->
                    if (success) {
                        showAddMealDialog = false
                        statusMessage = "Pasto pianificato con successo nel calendario!"
                    }
                }
            }
        )
    }
}

// ==========================================
// TAB 1: DISPENSA & SCORTE INVENTARIO
// ==========================================

@Composable
fun PantryInventoryTabContent(
    items: List<PantryItemDto>,
    isLoading: Boolean,
    onUpdateStock: (String, Double) -> Unit,
    onDelete: (String) -> Unit,
    onAddToShopping: (String, String) -> Unit,
    onOpenAddDialog: () -> Unit
) {
    var selectedCategory by remember { mutableStateOf("Tutti") }
    val categories = listOf("Tutti", "frigo", "dispensa", "ortofrutta", "freschi", "bevande", "surgelati", "condimenti")

    val lowStockCount = items.count { it.is_low_stock }
    val expiringCount = items.count { it.is_expiring_soon }

    val filteredItems = remember(items, selectedCategory) {
        if (selectedCategory == "Tutti") items
        else items.filter { it.category.equals(selectedCategory, ignoreCase = true) || it.location.equals(selectedCategory, ignoreCase = true) }
    }

    Scaffold(
        containerColor = Color.Transparent,
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = onOpenAddDialog,
                containerColor = IndigoPrimary,
                contentColor = Color.White,
                icon = { Text("+", fontSize = 20.sp, fontWeight = FontWeight.Bold) },
                text = { Text("Nuovo Prodotto", fontWeight = FontWeight.SemiBold) },
                modifier = Modifier.sizeIn(minWidth = 48.dp, minHeight = 48.dp)
            )
        }
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp)
        ) {
            item {
                Spacer(modifier = Modifier.height(6.dp))
                // KPI Cards
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    PantryKpiMiniCard(
                        title = "Totale Alimenti",
                        value = "${items.size}",
                        color = CyanAccent,
                        modifier = Modifier.weight(1f)
                    )
                    PantryKpiMiniCard(
                        title = "Sotto Soglia",
                        value = "$lowStockCount",
                        color = if (lowStockCount > 0) RoseDanger else Slate700,
                        modifier = Modifier.weight(1f)
                    )
                    PantryKpiMiniCard(
                        title = "In Scadenza",
                        value = "$expiringCount",
                        color = if (expiringCount > 0) Color(0xFFF59E0B) else Slate700,
                        modifier = Modifier.weight(1f)
                    )
                }
            }

            // Category Filter Pills
            item {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState())
                        .padding(vertical = 4.dp),
                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    categories.forEach { cat ->
                        Surface(
                            shape = RoundedCornerShape(16.dp),
                            color = if (selectedCategory == cat) CyanAccent else Slate800,
                            border = BorderStroke(1.dp, if (selectedCategory == cat) CyanAccent else Slate700),
                            modifier = Modifier
                                .clickable { selectedCategory = cat }
                                .semantics {
                                    contentDescription = "Filtro categoria $cat"
                                    role = Role.Button
                                }
                        ) {
                            Text(
                                text = cat.replaceFirstChar { it.uppercase() },
                                color = if (selectedCategory == cat) Color.Black else Color.White,
                                fontSize = 12.sp,
                                fontWeight = FontWeight.SemiBold,
                                modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp)
                            )
                        }
                    }
                }
            }

            if (filteredItems.isEmpty() && !isLoading) {
                item {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 40.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text("🥫", fontSize = 48.sp)
                        Spacer(modifier = Modifier.height(12.dp))
                        Text("Nessun alimento trovato in dispensa", color = Color.White, fontWeight = FontWeight.Bold)
                        Text("Aggiungi prodotti manualmente o scansiona uno scontrino", color = Color.White.copy(alpha = 0.6f), fontSize = 12.sp)
                    }
                }
            }

            // List of Pantry Items
            items(filteredItems, key = { it.id }) { item ->
                PantryItemCard(
                    item = item,
                    onIncrement = { onUpdateStock(item.id, 1.0) },
                    onDecrement = { onUpdateStock(item.id, -1.0) },
                    onDelete = { onDelete(item.id) },
                    onAddToShopping = { onAddToShopping(item.name, item.category) }
                )
            }

            item {
                Spacer(modifier = Modifier.height(80.dp))
            }
        }
    }
}

@Composable
fun PantryKpiMiniCard(
    title: String,
    value: String,
    color: Color,
    modifier: Modifier = Modifier
) {
    Surface(
        shape = RoundedCornerShape(12.dp),
        color = Slate800,
        border = BorderStroke(1.dp, color.copy(alpha = 0.4f)),
        modifier = modifier
    ) {
        Column(modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp)) {
            Text(title, fontSize = 10.sp, color = Color.White.copy(alpha = 0.65f), maxLines = 1)
            Spacer(modifier = Modifier.height(2.dp))
            Text(value, fontSize = 18.sp, fontWeight = FontWeight.Bold, color = color)
        }
    }
}

@Composable
fun PantryItemCard(
    item: PantryItemDto,
    onIncrement: () -> Unit,
    onDecrement: () -> Unit,
    onDelete: () -> Unit,
    onAddToShopping: () -> Unit
) {
    Surface(
        shape = RoundedCornerShape(14.dp),
        color = Slate800,
        border = BorderStroke(1.dp, if (item.is_low_stock) RoseDanger.copy(alpha = 0.5f) else Slate700),
        modifier = Modifier
            .fillMaxWidth()
            .semantics(mergeDescendants = true) {
                contentDescription = "Alimento ${item.name}, quantità %.1f %s, min %.1f".format(item.quantity, item.unit, item.min_threshold)
            }
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 14.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(item.name, color = Color.White, fontWeight = FontWeight.Bold, fontSize = 15.sp)
                    if (item.is_low_stock) {
                        Spacer(modifier = Modifier.width(6.dp))
                        Surface(
                            shape = RoundedCornerShape(6.dp),
                            color = RoseDanger.copy(alpha = 0.2f),
                            border = BorderStroke(1.dp, RoseDanger.copy(alpha = 0.6f))
                        ) {
                            Text(
                                "SOTTO SOGLIA",
                                color = RoseDanger,
                                fontSize = 9.sp,
                                fontWeight = FontWeight.Bold,
                                modifier = Modifier.padding(horizontal = 5.dp, vertical = 2.dp)
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.height(3.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "${item.category.uppercase()} • ${item.location}",
                        fontSize = 11.sp,
                        color = Color.White.copy(alpha = 0.6f)
                    )
                    item.expiry_date?.let { expiry ->
                        Text(
                            text = " • Scad. $expiry",
                            fontSize = 11.sp,
                            color = if (item.is_expiring_soon) Color(0xFFF59E0B) else Color.White.copy(alpha = 0.6f)
                        )
                    }
                }
            }

            // Quick Stock Actions (+ / -)
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                if (item.is_low_stock) {
                    IconButton(
                        onClick = onAddToShopping,
                        modifier = Modifier
                            .size(36.dp)
                            .background(CyanAccent.copy(alpha = 0.15f), CircleShape)
                            .semantics {
                                contentDescription = "Aggiungi ${item.name} alla spesa"
                                role = Role.Button
                            }
                    ) {
                        Text("🛒", fontSize = 14.sp)
                    }
                }

                IconButton(
                    onClick = onDecrement,
                    modifier = Modifier
                        .size(36.dp)
                        .background(Slate700, CircleShape)
                        .semantics {
                            contentDescription = "Riduci scorta di ${item.name}"
                            role = Role.Button
                        }
                ) {
                    Text("-", color = Color.White, fontWeight = FontWeight.Bold, fontSize = 18.sp)
                }

                Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.widthIn(min = 45.dp)) {
                    Text(
                        text = "%.1f".format(item.quantity).replace(".0", ""),
                        fontWeight = FontWeight.Bold,
                        color = if (item.is_low_stock) RoseDanger else Color.White,
                        fontSize = 15.sp
                    )
                    Text(item.unit, fontSize = 10.sp, color = Color.White.copy(alpha = 0.6f))
                }

                IconButton(
                    onClick = onIncrement,
                    modifier = Modifier
                        .size(36.dp)
                        .background(IndigoPrimary, CircleShape)
                        .semantics {
                            contentDescription = "Aumenta scorta di ${item.name}"
                            role = Role.Button
                        }
                ) {
                    Text("+", color = Color.White, fontWeight = FontWeight.Bold, fontSize = 18.sp)
                }
            }
        }
    }
}

// ==========================================
// TAB 2: RICETTARIO & PIANIFICAZIONE
// ==========================================

@Composable
fun PantryRecipesTabContent(
    recipes: List<RecipeDto>,
    isLoading: Boolean,
    onPlanMeal: (RecipeDto) -> Unit,
    onDelete: (String) -> Unit,
    onOpenAddDialog: () -> Unit
) {
    Scaffold(
        containerColor = Color.Transparent,
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = onOpenAddDialog,
                containerColor = EmeraldSuccess,
                contentColor = Color.White,
                icon = { Text("+", fontSize = 20.sp, fontWeight = FontWeight.Bold) },
                text = { Text("Nuova Ricetta", fontWeight = FontWeight.SemiBold) },
                modifier = Modifier.sizeIn(minWidth = 48.dp, minHeight = 48.dp)
            )
        }
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            if (recipes.isEmpty() && !isLoading) {
                item {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 40.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text("📖", fontSize = 48.sp)
                        Spacer(modifier = Modifier.height(12.dp))
                        Text("Ricettario vuoto", color = Color.White, fontWeight = FontWeight.Bold)
                        Text("Aggiungi la tua prima ricetta con ingredienti e porzioni", color = Color.White.copy(alpha = 0.6f), fontSize = 12.sp)
                    }
                }
            }

            items(recipes, key = { it.id }) { recipe ->
                RecipeCard(
                    recipe = recipe,
                    onPlan = { onPlanMeal(recipe) },
                    onDelete = { onDelete(recipe.id) }
                )
            }

            item {
                Spacer(modifier = Modifier.height(80.dp))
            }
        }
    }
}

@Composable
fun RecipeCard(
    recipe: RecipeDto,
    onPlan: () -> Unit,
    onDelete: () -> Unit
) {
    Surface(
        shape = RoundedCornerShape(14.dp),
        color = Slate800,
        border = BorderStroke(1.dp, Slate700),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(14.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column {
                    Text(recipe.title, color = Color.White, fontWeight = FontWeight.Bold, fontSize = 16.sp)
                    Text(
                        "${recipe.category.uppercase()} • ${recipe.portions_base} porzioni base • ⏱️ ${recipe.prep_time_min + recipe.cook_time_min} min",
                        color = CyanAccent,
                        fontSize = 11.sp
                    )
                }

                Button(
                    onClick = onPlan,
                    colors = ButtonDefaults.buttonColors(containerColor = IndigoPrimary),
                    shape = RoundedCornerShape(8.dp),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp),
                    modifier = Modifier.heightIn(min = 40.dp)
                ) {
                    Text("Pianifica", fontSize = 12.sp, fontWeight = FontWeight.Bold)
                }
            }

            if (recipe.description.isNotBlank()) {
                Spacer(modifier = Modifier.height(6.dp))
                Text(recipe.description, color = Color.White.copy(alpha = 0.75f), fontSize = 13.sp)
            }

            Spacer(modifier = Modifier.height(10.dp))
            Text("Ingredienti:", color = Color.White.copy(alpha = 0.6f), fontSize = 11.sp, fontWeight = FontWeight.SemiBold)
            Spacer(modifier = Modifier.height(4.dp))
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .horizontalScroll(rememberScrollState()),
                horizontalArrangement = Arrangement.spacedBy(6.dp)
            ) {
                recipe.ingredients.forEach { ing ->
                    Surface(
                        shape = RoundedCornerShape(6.dp),
                        color = Slate700.copy(alpha = 0.5f),
                        border = BorderStroke(1.dp, Slate700)
                    ) {
                        Text(
                            text = "${ing.item_name} (%.0f %s)".format(ing.quantity, ing.unit),
                            color = Color.White.copy(alpha = 0.9f),
                            fontSize = 11.sp,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                        )
                    }
                }
            }
        }
    }
}

// ==========================================
// TAB 3: CALENDARIO PASTI & CONSUMO DISPENSA
// ==========================================

@Composable
fun PantryMealCalendarTabContent(
    mealPlans: List<MealPlanDto>,
    recipes: List<RecipeDto>,
    isLoading: Boolean,
    onConsumeMeal: (String, Int) -> Unit,
    onDeleteMeal: (String) -> Unit,
    onOpenAddDialog: () -> Unit
) {
    Scaffold(
        containerColor = Color.Transparent,
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = onOpenAddDialog,
                containerColor = CyanAccent,
                contentColor = Color.Black,
                icon = { Text("+", fontSize = 20.sp, fontWeight = FontWeight.Bold) },
                text = { Text("Pianifica Pasto", fontWeight = FontWeight.Bold) },
                modifier = Modifier.sizeIn(minWidth = 48.dp, minHeight = 48.dp)
            )
        }
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            item {
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = CyanAccent.copy(alpha = 0.12f),
                    border = BorderStroke(1.dp, CyanAccent.copy(alpha = 0.35f)),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(modifier = Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text("⚡", fontSize = 20.sp)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            "Cliccando 'Consuma Pasto', le scorte in dispensa vengono scalate in tempo reale in base al numero di commensali!",
                            color = CyanAccent,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold
                        )
                    }
                }
            }

            if (mealPlans.isEmpty() && !isLoading) {
                item {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 40.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text("📅", fontSize = 48.sp)
                        Spacer(modifier = Modifier.height(12.dp))
                        Text("Nessun pasto pianificato", color = Color.White, fontWeight = FontWeight.Bold)
                        Text("Pianifica colazione, pranzo, cena o spuntino nel calendario", color = Color.White.copy(alpha = 0.6f), fontSize = 12.sp)
                    }
                }
            }

            items(mealPlans, key = { it.id }) { plan ->
                MealPlanCard(
                    plan = plan,
                    onConsume = { persons -> onConsumeMeal(plan.id, persons) },
                    onDelete = { onDeleteMeal(plan.id) }
                )
            }

            item {
                Spacer(modifier = Modifier.height(80.dp))
            }
        }
    }
}

@Composable
fun MealPlanCard(
    plan: MealPlanDto,
    onConsume: (Int) -> Unit,
    onDelete: () -> Unit
) {
    var persons by remember { mutableStateOf(plan.persons_count) }

    Surface(
        shape = RoundedCornerShape(14.dp),
        color = Slate800,
        border = BorderStroke(1.dp, if (plan.is_consumed) EmeraldSuccess.copy(alpha = 0.4f) else Slate700),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(14.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    val icon = when (plan.meal_type) {
                        "breakfast" -> "☕"
                        "lunch" -> "🍝"
                        "dinner" -> "🍲"
                        "snack" -> "🍎"
                        else -> "🍰"
                    }
                    Text(icon, fontSize = 22.sp)
                    Spacer(modifier = Modifier.width(8.dp))
                    Column {
                        Text(
                            text = plan.recipe_title.ifBlank { "Pasto Personalizzato" },
                            color = Color.White,
                            fontWeight = FontWeight.Bold,
                            fontSize = 15.sp
                        )
                        Text(
                            text = "${plan.date} • ${plan.meal_type.uppercase()}",
                            color = CyanAccent,
                            fontSize = 11.sp
                        )
                    }
                }

                if (plan.is_consumed) {
                    Surface(
                        shape = RoundedCornerShape(8.dp),
                        color = EmeraldSuccess.copy(alpha = 0.2f),
                        border = BorderStroke(1.dp, EmeraldSuccess)
                    ) {
                        Text(
                            "CONSUMATO ✓",
                            color = EmeraldSuccess,
                            fontSize = 10.sp,
                            fontWeight = FontWeight.Bold,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                        )
                    }
                } else {
                    IconButton(onClick = onDelete, modifier = Modifier.size(32.dp)) {
                        Text("🗑️", fontSize = 14.sp)
                    }
                }
            }

            if (plan.custom_notes.isNotBlank()) {
                Spacer(modifier = Modifier.height(6.dp))
                Text(plan.custom_notes, color = Color.White.copy(alpha = 0.7f), fontSize = 12.sp)
            }

            Spacer(modifier = Modifier.height(10.dp))

            // Action row: Persons selector & "Consuma" action
            if (!plan.is_consumed) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("Commensali:", color = Color.White.copy(alpha = 0.7f), fontSize = 12.sp)
                        Spacer(modifier = Modifier.width(6.dp))
                        IconButton(
                            onClick = { if (persons > 1) persons-- },
                            modifier = Modifier
                                .size(32.dp)
                                .background(Slate700, CircleShape)
                        ) {
                            Text("-", color = Color.White, fontWeight = FontWeight.Bold)
                        }
                        Text(
                            "$persons",
                            color = Color.White,
                            fontWeight = FontWeight.Bold,
                            modifier = Modifier.padding(horizontal = 8.dp)
                        )
                        IconButton(
                            onClick = { persons++ },
                            modifier = Modifier
                                .size(32.dp)
                                .background(Slate700, CircleShape)
                        ) {
                            Text("+", color = Color.White, fontWeight = FontWeight.Bold)
                        }
                    }

                    Button(
                        onClick = { onConsume(persons) },
                        colors = ButtonDefaults.buttonColors(containerColor = EmeraldSuccess),
                        shape = RoundedCornerShape(8.dp),
                        modifier = Modifier.heightIn(min = 40.dp)
                    ) {
                        Text("Consuma Pasto 🍽️", color = Color.Black, fontWeight = FontWeight.Bold, fontSize = 12.sp)
                    }
                }
            }
        }
    }
}

// ==========================================
// TAB 4: SCANNER SCONTRINO (RECEIPT OCR)
// ==========================================

@Composable
fun PantryReceiptScannerTabContent(
    viewModel: DashboardViewModel,
    onImportSuccess: (Int, Double) -> Unit
) {
    var rawText by remember { mutableStateOf("") }
    val parsedReceipt by viewModel.parsedReceipt.collectAsState()
    val isLoading by viewModel.isPantryLoading.collectAsState()

    var autoAddToFinance by remember { mutableStateOf(true) }
    var autoUpdatePantry by remember { mutableStateOf(true) }

    val sampleEsselunga = """
        ESSELUNGA S.P.A.
        VIA CASILINA 100 - MILANO
        DATA: 17/10/2026 ORE: 18:30

        PASTA BARILLA 500G        1.20
        LATTE INTERO 1L           1.50
        MELE FUJI KG 1.500        3.45
        CAFFE LAVAZZA 250G        3.80
        PETTO DI POLLO KG 0.800   7.60

        TOTALE COMPLESSIVO       17.55
        PAGAMENTO: CARTA CREDITO
    """.trimIndent()

    val sampleConad = """
        CONAD CITY - ROMA
        17/10/2026
        UOVA BIO 6PZ              2.40
        OLIO EVO 1L               7.90
        PARMIGIANO REGGIANO 300G  5.50
        BANANE KG 1.200           1.90
        TOTALE EURO              17.70
    """.trimIndent()

    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        item {
            Text(
                "Lettore Scontrini Intelligente",
                fontWeight = FontWeight.Bold,
                fontSize = 16.sp,
                color = Color.White
            )
            Text(
                "Incolla o acquisisci il testo dello scontrino per estrarre articoli, pesi, prezzi e registrare la spesa.",
                fontSize = 12.sp,
                color = Color.White.copy(alpha = 0.7f)
            )
        }

        // Demo sample buttons
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = { rawText = sampleEsselunga },
                    colors = ButtonDefaults.buttonColors(containerColor = Slate800),
                    border = BorderStroke(1.dp, Slate700)
                ) {
                    Text("Campione Esselunga", fontSize = 11.sp, color = CyanAccent)
                }
                Button(
                    onClick = { rawText = sampleConad },
                    colors = ButtonDefaults.buttonColors(containerColor = Slate800),
                    border = BorderStroke(1.dp, Slate700)
                ) {
                    Text("Campione Conad", fontSize = 11.sp, color = CyanAccent)
                }
            }
        }

        // Raw Text Input
        item {
            OutlinedTextField(
                value = rawText,
                onValueChange = { rawText = it },
                label = { Text("Testo OCR Scontrino") },
                placeholder = { Text("Incolla qui le righe dello scontrino...") },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(130.dp),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedTextColor = Color.White,
                    unfocusedTextColor = Color.White,
                    focusedBorderColor = CyanAccent,
                    unfocusedBorderColor = Slate700
                )
            )
        }

        item {
            Button(
                onClick = {
                    if (rawText.isNotBlank()) {
                        viewModel.parseReceipt(rawText)
                    }
                },
                enabled = rawText.isNotBlank() && !isLoading,
                colors = ButtonDefaults.buttonColors(containerColor = IndigoPrimary),
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = 48.dp),
                shape = RoundedCornerShape(10.dp)
            ) {
                Text(
                    if (isLoading) "Analisi in corso..." else "Analizza Scontrino 🔍",
                    fontWeight = FontWeight.Bold,
                    fontSize = 14.sp
                )
            }
        }

        // Parsed Receipt Details
        parsedReceipt?.let { receipt ->
            item {
                Surface(
                    shape = RoundedCornerShape(14.dp),
                    color = Slate800,
                    border = BorderStroke(1.dp, CyanAccent.copy(alpha = 0.4f)),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Column(modifier = Modifier.padding(14.dp)) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Column {
                                Text(
                                    receipt.store_name.ifBlank { "Supermercato" },
                                    fontWeight = FontWeight.Bold,
                                    color = Color.White,
                                    fontSize = 16.sp
                                )
                                Text(receipt.date, color = Color.White.copy(alpha = 0.6f), fontSize = 11.sp)
                            }
                            Column(horizontalAlignment = Alignment.End) {
                                Text("Totale Speso", color = CyanAccent, fontSize = 11.sp)
                                Text("€%.2f".format(receipt.total_amount), fontWeight = FontWeight.Bold, fontSize = 18.sp, color = Color.White)
                            }
                        }

                        Spacer(modifier = Modifier.height(12.dp))
                        Text("Prodotti Rilevati (${receipt.items.size}):", fontWeight = FontWeight.SemiBold, fontSize = 12.sp, color = Color.White)
                        Spacer(modifier = Modifier.height(6.dp))

                        receipt.items.forEach { item ->
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(vertical = 3.dp),
                                horizontalArrangement = Arrangement.SpaceBetween
                            ) {
                                Text(
                                    "${item.item_name} (%.1f %s)".format(item.quantity, item.unit),
                                    color = Color.White.copy(alpha = 0.85f),
                                    fontSize = 12.sp,
                                    modifier = Modifier.weight(1f)
                                )
                                Text("€%.2f".format(item.total_price), color = EmeraldSuccess, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                            }
                        }

                        Spacer(modifier = Modifier.height(14.dp))

                        // Toggles for Auto-actions
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Text("Aggiorna Scorte Dispensa", color = Color.White, fontSize = 13.sp)
                            Switch(
                                checked = autoUpdatePantry,
                                onCheckedChange = { autoUpdatePantry = it },
                                colors = SwitchDefaults.colors(checkedThumbColor = CyanAccent)
                            )
                        }

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Text("Registra in Bilancio Casa (€)", color = Color.White, fontSize = 13.sp)
                            Switch(
                                checked = autoAddToFinance,
                                onCheckedChange = { autoAddToFinance = it },
                                colors = SwitchDefaults.colors(checkedThumbColor = EmeraldSuccess)
                            )
                        }

                        Spacer(modifier = Modifier.height(10.dp))

                        Button(
                            onClick = {
                                viewModel.importReceipt(receipt, autoAddToFinance, autoUpdatePantry) { res ->
                                    if (res != null) {
                                        onImportSuccess(res.imported_items_count, res.total_amount)
                                    }
                                }
                            },
                            enabled = !isLoading,
                            colors = ButtonDefaults.buttonColors(containerColor = EmeraldSuccess),
                            modifier = Modifier
                                .fillMaxWidth()
                                .heightIn(min = 48.dp),
                            shape = RoundedCornerShape(10.dp)
                        ) {
                            Text(
                                "Conferma e Importa in Dispensa ✓",
                                color = Color.Black,
                                fontWeight = FontWeight.Bold,
                                fontSize = 14.sp
                            )
                        }
                    }
                }
            }
        }

        item {
            Spacer(modifier = Modifier.height(40.dp))
        }
    }
}

// ==========================================
// DIALOGS
// ==========================================

@Composable
fun AddPantryItemDialog(
    onDismiss: () -> Unit,
    onConfirm: (CreatePantryItemRequestDto) -> Unit
) {
    var name by remember { mutableStateOf("") }
    var category by remember { mutableStateOf("dispensa") }
    var location by remember { mutableStateOf("dispensa") }
    var quantityStr by remember { mutableStateOf("1.0") }
    var unit by remember { mutableStateOf("pz") }
    var minThresholdStr by remember { mutableStateOf("1.0") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Nuovo Alimento in Dispensa", color = Color.White) },
        containerColor = Slate800,
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text("Nome Prodotto") },
                    modifier = Modifier.fillMaxWidth()
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = category,
                        onValueChange = { category = it },
                        label = { Text("Categoria") },
                        modifier = Modifier.weight(1f)
                    )
                    OutlinedTextField(
                        value = location,
                        onValueChange = { location = it },
                        label = { Text("Posizione") },
                        modifier = Modifier.weight(1f)
                    )
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = quantityStr,
                        onValueChange = { quantityStr = it },
                        label = { Text("Quantità") },
                        modifier = Modifier.weight(1f)
                    )
                    OutlinedTextField(
                        value = unit,
                        onValueChange = { unit = it },
                        label = { Text("Unità") },
                        modifier = Modifier.weight(1f)
                    )
                }
                OutlinedTextField(
                    value = minThresholdStr,
                    onValueChange = { minThresholdStr = it },
                    label = { Text("Soglia Minima Allerta") },
                    modifier = Modifier.fillMaxWidth()
                )
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    val q = quantityStr.toDoubleOrNull() ?: 1.0
                    val th = minThresholdStr.toDoubleOrNull() ?: 1.0
                    onConfirm(
                        CreatePantryItemRequestDto(
                            name = name,
                            category = category,
                            location = location,
                            quantity = q,
                            unit = unit,
                            min_threshold = th
                        )
                    )
                },
                enabled = name.isNotBlank(),
                colors = ButtonDefaults.buttonColors(containerColor = IndigoPrimary)
            ) {
                Text("Salva")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Annulla", color = Color.White.copy(alpha = 0.7f))
            }
        }
    )
}

@Composable
fun AddRecipeDialog(
    pantryItems: List<PantryItemDto>,
    onDismiss: () -> Unit,
    onConfirm: (CreateRecipeRequestDto) -> Unit
) {
    var title by remember { mutableStateOf("") }
    var description by remember { mutableStateOf("") }
    var portionsStr by remember { mutableStateOf("2") }
    var prepTimeStr by remember { mutableStateOf("10") }
    var cookTimeStr by remember { mutableStateOf("15") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Nuova Ricetta", color = Color.White) },
        containerColor = Slate800,
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = title,
                    onValueChange = { title = it },
                    label = { Text("Titolo Ricetta") },
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedTextField(
                    value = description,
                    onValueChange = { description = it },
                    label = { Text("Descrizione") },
                    modifier = Modifier.fillMaxWidth()
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = portionsStr,
                        onValueChange = { portionsStr = it },
                        label = { Text("Porzioni") },
                        modifier = Modifier.weight(1f)
                    )
                    OutlinedTextField(
                        value = prepTimeStr,
                        onValueChange = { prepTimeStr = it },
                        label = { Text("Prep (min)") },
                        modifier = Modifier.weight(1f)
                    )
                    OutlinedTextField(
                        value = cookTimeStr,
                        onValueChange = { cookTimeStr = it },
                        label = { Text("Cottura (min)") },
                        modifier = Modifier.weight(1f)
                    )
                }
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    onConfirm(
                        CreateRecipeRequestDto(
                            title = title,
                            description = description,
                            portions_base = portionsStr.toIntOrNull() ?: 2,
                            prep_time_min = prepTimeStr.toIntOrNull() ?: 10,
                            cook_time_min = cookTimeStr.toIntOrNull() ?: 15
                        )
                    )
                },
                enabled = title.isNotBlank(),
                colors = ButtonDefaults.buttonColors(containerColor = EmeraldSuccess)
            ) {
                Text("Crea Ricetta")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Annulla", color = Color.White.copy(alpha = 0.7f))
            }
        }
    )
}

@Composable
fun AddMealPlanDialog(
    preselectedRecipe: RecipeDto?,
    recipes: List<RecipeDto>,
    onDismiss: () -> Unit,
    onConfirm: (CreateMealPlanRequestDto) -> Unit
) {
    var selectedRecipeId by remember { mutableStateOf(preselectedRecipe?.id ?: "") }
    var mealType by remember { mutableStateOf("lunch") }
    var dateStr by remember { mutableStateOf(LocalDate.now().toString()) }
    var personsStr by remember { mutableStateOf("${preselectedRecipe?.portions_base ?: 2}") }
    var customNotes by remember { mutableStateOf("") }

    val mealTypes = listOf("breakfast" to "☕ Colazione", "lunch" to "🍝 Pranzo", "snack" to "🍎 Spuntino", "dinner" to "🍲 Cena", "dessert" to "🍰 Dessert")

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Pianifica Pasto", color = Color.White) },
        containerColor = Slate800,
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedTextField(
                    value = dateStr,
                    onValueChange = { dateStr = it },
                    label = { Text("Data (YYYY-MM-DD)") },
                    modifier = Modifier.fillMaxWidth()
                )

                Text("Tipo di Pasto:", color = Color.White.copy(alpha = 0.7f), fontSize = 12.sp)
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    mealTypes.forEach { (type, label) ->
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = if (mealType == type) CyanAccent else Slate700,
                            modifier = Modifier.clickable { mealType = type }
                        ) {
                            Text(
                                label,
                                color = if (mealType == type) Color.Black else Color.White,
                                fontSize = 11.sp,
                                fontWeight = FontWeight.Bold,
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 6.dp)
                            )
                        }
                    }
                }

                OutlinedTextField(
                    value = personsStr,
                    onValueChange = { personsStr = it },
                    label = { Text("Numero Commensali") },
                    modifier = Modifier.fillMaxWidth()
                )

                OutlinedTextField(
                    value = customNotes,
                    onValueChange = { customNotes = it },
                    label = { Text("Note pasto") },
                    modifier = Modifier.fillMaxWidth()
                )
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    onConfirm(
                        CreateMealPlanRequestDto(
                            date = dateStr,
                            meal_type = mealType,
                            recipe_id = selectedRecipeId.ifBlank { null },
                            custom_notes = customNotes,
                            persons_count = personsStr.toIntOrNull() ?: 2
                        )
                    )
                },
                colors = ButtonDefaults.buttonColors(containerColor = CyanAccent)
            ) {
                Text("Salva Pasto", color = Color.Black, fontWeight = FontWeight.Bold)
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Annulla", color = Color.White.copy(alpha = 0.7f))
            }
        }
    )
}
