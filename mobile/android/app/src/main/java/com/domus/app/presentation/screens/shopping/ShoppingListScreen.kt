package com.domus.app.presentation.screens.shopping

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateColorAsState
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.RoseDanger
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.theme.Slate900
import com.domus.app.presentation.viewmodel.DashboardViewModel

data class ShoppingItem(
    val id: String,
    val name: String,
    val category: String,
    val isCompleted: Boolean = false
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ShoppingListScreen(
    viewModel: DashboardViewModel? = null,
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    LaunchedEffect(Unit) {
        viewModel?.loadShopping()
    }

    val serverItems by viewModel?.shoppingItems?.collectAsState() ?: remember { mutableStateOf(emptyList()) }

    val items = remember(serverItems) {
        serverItems.map { dto ->
            ShoppingItem(
                id = dto.id,
                name = dto.name,
                category = dto.category.ifBlank { "Dispensa & Colazione" },
                isCompleted = dto.completed
            )
        }
    }

    var newItemName by remember { mutableStateOf("") }
    var selectedCategory by remember { mutableStateOf("Dispensa & Colazione") }

    val categories = listOf("Ortofrutta", "Latticini & Freschi", "Dispensa & Colazione", "Cura Casa & Pulizia")

    val completedCount = items.count { it.isCompleted }
    val totalCount = items.size

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(Slate900)
    ) {
        // Top Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            IconButton(onClick = onDismiss) {
                Text("✕", fontSize = 20.sp, color = Color.White)
            }

            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = "Lista della Spesa",
                    fontWeight = FontWeight.Bold,
                    fontSize = 18.sp,
                    color = Color.White
                )
                Text(
                    text = "$completedCount di $totalCount completati",
                    fontSize = 12.sp,
                    color = CyanAccent
                )
            }

            // Clear completed items button
            IconButton(
                onClick = { viewModel?.clearCompletedShopping() },
                enabled = completedCount > 0
            ) {
                Text("🗑️", fontSize = 18.sp)
            }
        }

        // Add Item Bar
        Card(
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = Slate800),
            border = BorderStroke(1.dp, Slate700),
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 6.dp)
        ) {
            Column(modifier = Modifier.padding(12.dp)) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    OutlinedTextField(
                        value = newItemName,
                        onValueChange = { newItemName = it },
                        placeholder = { Text("Aggiungi prodotto...", fontSize = 14.sp) },
                        singleLine = true,
                        shape = RoundedCornerShape(12.dp),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedBorderColor = CyanAccent,
                            unfocusedBorderColor = Slate700
                        ),
                        modifier = Modifier.weight(1f)
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    Button(
                        onClick = {
                            if (newItemName.isNotBlank()) {
                                viewModel?.createShopping(newItemName.trim(), selectedCategory, "1")
                                newItemName = ""
                            }
                        },
                        enabled = newItemName.isNotBlank(),
                        shape = RoundedCornerShape(12.dp),
                        colors = ButtonDefaults.buttonColors(containerColor = CyanAccent)
                    ) {
                        Text("+", color = Color.Black, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                    }
                }

                Spacer(modifier = Modifier.height(8.dp))

                // Category Selector Chips
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    categories.forEach { cat ->
                        val isSelected = selectedCategory == cat
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = if (isSelected) IndigoPrimary else Slate900,
                            border = BorderStroke(1.dp, if (isSelected) CyanAccent else Slate700),
                            modifier = Modifier
                                .weight(1f)
                                .clickable { selectedCategory = cat }
                        ) {
                            Text(
                                text = cat.split(" ")[0],
                                fontSize = 11.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = if (isSelected) Color.White else Color.White.copy(alpha = 0.6f),
                                modifier = Modifier.padding(vertical = 6.dp, horizontal = 4.dp),
                                maxLines = 1
                            )
                        }
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        // Grouped Items by Category
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
            contentPadding = PaddingValues(bottom = 24.dp)
        ) {
            categories.forEach { category ->
                val categoryItems = items.filter { it.category == category }
                if (categoryItems.isNotEmpty()) {
                    item(key = "header_$category") {
                        Text(
                            text = category.uppercase(),
                            fontSize = 11.sp,
                            fontWeight = FontWeight.Bold,
                            color = CyanAccent,
                            modifier = Modifier.padding(top = 10.dp, bottom = 4.dp)
                        )
                    }

                    items(categoryItems, key = { it.id }) { item ->
                        val itemColor by animateColorAsState(
                            targetValue = if (item.isCompleted) Slate800.copy(alpha = 0.5f) else Slate800,
                            label = "item_color"
                        )

                        Card(
                            shape = RoundedCornerShape(14.dp),
                            colors = CardDefaults.cardColors(containerColor = itemColor),
                            border = BorderStroke(
                                1.dp,
                                if (item.isCompleted) Color.Transparent else Slate700
                            ),
                            modifier = Modifier
                                .fillMaxWidth()
                                .semantics {
                                    role = Role.Checkbox
                                    contentDescription = "${item.name}, ${if (item.isCompleted) "acquistato" else "da acquistare"}"
                                }
                                .clickable {
                                    viewModel?.toggleShopping(item.id)
                                }
                        ) {
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = 14.dp, vertical = 12.dp),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                // Checkbox icon
                                Surface(
                                    shape = CircleShape,
                                    color = if (item.isCompleted) EmeraldSuccess else Color.Transparent,
                                    border = BorderStroke(
                                        2.dp,
                                        if (item.isCompleted) EmeraldSuccess else Color.White.copy(alpha = 0.5f)
                                    ),
                                    modifier = Modifier.size(24.dp)
                                ) {
                                    if (item.isCompleted) {
                                        Box(contentAlignment = Alignment.Center) {
                                            Text("✓", color = Color.White, fontSize = 13.sp, fontWeight = FontWeight.Bold)
                                        }
                                    }
                                }

                                Spacer(modifier = Modifier.width(12.dp))

                                Text(
                                    text = item.name,
                                    fontSize = 15.sp,
                                    fontWeight = if (item.isCompleted) FontWeight.Normal else FontWeight.SemiBold,
                                    color = if (item.isCompleted) Color.White.copy(alpha = 0.4f) else Color.White,
                                    textDecoration = if (item.isCompleted) TextDecoration.LineThrough else TextDecoration.None,
                                    modifier = Modifier.weight(1f)
                                )

                                // Delete button
                                IconButton(
                                    onClick = { viewModel?.deleteShopping(item.id) },
                                    modifier = Modifier.size(28.dp)
                                ) {
                                    Text("✕", color = Color.White.copy(alpha = 0.4f), fontSize = 14.sp)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
