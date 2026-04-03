using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Reflection;
using System.Text.Json;
using Microsoft.AspNetCore.Mvc;

namespace SysBot.Pokemon.Web.Api;

/// <summary>
/// Read and update program configuration, including the full PokeTradeHubConfig.
/// Also exposes a schema endpoint for dynamic UI generation.
/// </summary>
[ApiController]
[Route("api/config")]
public class ConfigController(IPokeBotRunner runner, ProgramConfig programConfig) : ControllerBase
{
    /// <summary>GET /api/config — return the full ProgramConfig (source-generated serialization).</summary>
    [HttpGet]
    public IActionResult GetConfig()
    {
        return new JsonResult(programConfig, ProgramConfigContext.Default.ProgramConfig);
    }

    /// <summary>GET /api/config/hub — return the PokeTradeHubConfig owned by the runner.</summary>
    [HttpGet("hub")]
    public IActionResult GetHub()
    {
        return Ok(runner.Config);
    }

    /// <summary>
    /// PATCH /api/config/hub — partial update of PokeTradeHubConfig.
    /// Walks the incoming JSON properties and applies them via reflection.
    /// </summary>
    [HttpPatch("hub")]
    public IActionResult PatchHub([FromBody] JsonElement body)
    {
        if (body.ValueKind != JsonValueKind.Object)
            return BadRequest(new { error = "Request body must be a JSON object." });

        try
        {
            ApplyJsonToObject(body, runner.Config);
        }
        catch (Exception ex)
        {
            return BadRequest(new { error = ex.Message });
        }

        return Ok(runner.Config);
    }

    /// <summary>
    /// GET /api/config/hub/schema — introspect PokeTradeHubConfig and return a JSON schema
    /// grouped by [Category], with type info, descriptions, and enum values.
    /// </summary>
    [HttpGet("hub/schema")]
    public IActionResult GetHubSchema()
    {
        var schema = BuildCategorizedSchema(typeof(PokeTradeHubConfig), runner.Config);
        return Ok(schema);
    }

    // ── Reflection helpers: patch ───────────────────────────────────────

    /// <summary>Walk JSON properties and set matching properties on <paramref name="target"/>.</summary>
    private static void ApplyJsonToObject(JsonElement json, object target)
    {
        var type = target.GetType();

        foreach (var prop in json.EnumerateObject())
        {
            var pi = type.GetProperty(prop.Name, BindingFlags.Public | BindingFlags.Instance | BindingFlags.IgnoreCase);
            if (pi is null || !pi.CanWrite)
                continue;

            var propType = pi.PropertyType;

            // Nested object — recurse into the existing instance.
            if (prop.Value.ValueKind == JsonValueKind.Object && !propType.IsEnum && !propType.IsPrimitive)
            {
                var existing = pi.GetValue(target);
                if (existing is not null)
                {
                    ApplyJsonToObject(prop.Value, existing);
                    continue;
                }
            }

            // Scalar / enum / array — deserialize directly.
            var value = DeserializeValue(prop.Value, propType);
            pi.SetValue(target, value);
        }
    }

    /// <summary>Convert a <see cref="JsonElement"/> to a CLR value of the given <paramref name="targetType"/>.</summary>
    private static object? DeserializeValue(JsonElement element, Type targetType)
    {
        // Enum stored as string.
        if (targetType.IsEnum)
        {
            var raw = element.GetString();
            if (raw is null)
                throw new InvalidOperationException($"Cannot parse null as {targetType.Name}.");
            return Enum.Parse(targetType, raw, ignoreCase: true);
        }

        // Primitive / well-known types.
        return element.Deserialize(targetType);
    }

    // ── Reflection helpers: schema ──────────────────────────────────────

    /// <summary>
    /// Build a categorized schema for the top-level hub config.
    /// Returns { categories: { "CategoryName": { "PropertyName": { type, description, ... } } } }
    /// </summary>
    private static Dictionary<string, object> BuildCategorizedSchema(Type type, object? instance)
    {
        var categories = new Dictionary<string, Dictionary<string, object>>(StringComparer.OrdinalIgnoreCase);

        foreach (var pi in type.GetProperties(BindingFlags.Public | BindingFlags.Instance))
        {
            // Skip indexed properties (e.g., this[int index]) — they require parameters to access.
            if (pi.GetIndexParameters().Length != 0)
                continue;

            var browsable = pi.GetCustomAttribute<BrowsableAttribute>();
            if (browsable is { Browsable: false })
                continue;

            var category = pi.GetCustomAttribute<CategoryAttribute>()?.Category ?? "General";
            var description = pi.GetCustomAttribute<DescriptionAttribute>()?.Description;
            var value = instance is not null ? pi.GetValue(instance) : null;

            var propSchema = BuildPropertySchema(pi.PropertyType, description, value);

            if (!categories.TryGetValue(category, out var section))
            {
                section = [];
                categories[category] = section;
            }
            section[pi.Name] = propSchema;
        }

        return new Dictionary<string, object> { ["categories"] = categories };
    }

    /// <summary>
    /// Build a flat property map for nested objects (no categories).
    /// Returns { "PropertyName": { type, description, ... } }
    /// </summary>
    private static Dictionary<string, object> BuildFlatProperties(Type type, object? instance, int depth = 0)
    {
        var props = new Dictionary<string, object>();
        if (depth > 3) return props; // Prevent excessive recursion

        foreach (var pi in type.GetProperties(BindingFlags.Public | BindingFlags.Instance))
        {
            if (pi.GetIndexParameters().Length != 0)
                continue;

            var browsable = pi.GetCustomAttribute<BrowsableAttribute>();
            if (browsable is { Browsable: false })
                continue;

            // Skip collection/array types — they're not simple settings
            if (pi.PropertyType.IsArray || (pi.PropertyType.IsGenericType &&
                pi.PropertyType.GetGenericTypeDefinition().GetInterfaces()
                    .Any(i => i.IsGenericType && i.GetGenericTypeDefinition() == typeof(IEnumerable<>))))
                continue;

            var description = pi.GetCustomAttribute<DescriptionAttribute>()?.Description;
            object? value = null;
            try { value = instance is not null ? pi.GetValue(instance) : null; }
            catch { continue; } // Skip properties that throw on access

            props[pi.Name] = BuildPropertySchema(pi.PropertyType, description, value, depth);
        }

        return props;
    }

    /// <summary>Build a schema entry for a single property type.</summary>
    private static Dictionary<string, object> BuildPropertySchema(Type propType, string? description, object? value, int depth = 0)
    {
        var schema = new Dictionary<string, object>();

        if (description is not null)
            schema["description"] = description;

        if (propType.IsEnum)
        {
            schema["type"] = "enum";
            schema["enumValues"] = Enum.GetNames(propType);
            if (value is not null) schema["value"] = value.ToString()!;
        }
        else if (propType == typeof(bool))
        {
            schema["type"] = "boolean";
            if (value is not null) schema["value"] = value;
        }
        else if (propType == typeof(int) || propType == typeof(long) || propType == typeof(short) || propType == typeof(byte))
        {
            schema["type"] = "integer";
            if (value is not null) schema["value"] = value;
        }
        else if (propType == typeof(float) || propType == typeof(double) || propType == typeof(decimal))
        {
            schema["type"] = "number";
            if (value is not null) schema["value"] = value;
        }
        else if (propType == typeof(string))
        {
            schema["type"] = "string";
            if (value is not null) schema["value"] = value;
        }
        else if (propType.IsClass && propType != typeof(string))
        {
            schema["type"] = "object";
            if (description is not null) schema["description"] = description;
            schema["properties"] = BuildFlatProperties(propType, value, depth + 1);
        }
        else
        {
            schema["type"] = propType.Name.ToLowerInvariant();
            if (value is not null) schema["value"] = value;
        }

        return schema;
    }
}
