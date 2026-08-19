# Figma MCP Android

*[English](README.md) · **Tiếng Việt***

MCP server viết bằng Go, giao tiếp qua stdio, giúp các AI client đọc trực tiếp file Figma đang mở thông qua một plugin chạy trên máy bạn — không cần Figma API token. Server tập trung vào việc đọc và phân tích thiết kế, xuất tài nguyên, đồng thời có sẵn công cụ chuyển SVG sang Android VectorDrawable cho anh em làm Android.

Repo gồm hai phần hoạt động cùng nhau:

- MCP server cung cấp các tool qua stdio
- plugin Figma Desktop kết nối tới server tại `ws://127.0.0.1:1994` (mặc định)

## Features

- Cho phép AI client đọc trực tiếp file Figma đang mở
- Không cần Figma REST API key
- Plugin kết nối nội bộ qua WebSocket, dữ liệu không đi ra ngoài
- 21 tool để duyệt tài liệu, xem metadata, tìm kiếm, xuất file, chụp ảnh và chuyển đổi tài nguyên Android
- Xuất design token
- Chụp và xuất ảnh, có thể lưu thẳng thành file
- Chuyển đổi Android VectorDrawable bằng `convert_svg_to_android_drawable`
- Server Go kèm gói npm để MCP client chạy trực tiếp

## Yêu cầu

- Node.js `>=18`
- Figma Desktop
- Go `1.26.1+` nếu muốn phát triển server
- Bun nếu muốn phát triển plugin

## Cài đặt

### 1. Cấu hình AI client

#### Claude Code / Claude Desktop / VS Code / Cursor / Antigravity

Nội dung khai báo giống hệt nhau, chỉ khác chỗ đặt file:

- Claude Code — `.mcp.json` ở thư mục gốc của project
- Claude Desktop — `claude_desktop_config.json`
- Cursor — `.cursor/mcp.json`
- VS Code — `.vscode/mcp.json` (VS Code dùng key ngoài cùng là `servers` chứ không phải `mcpServers`)
- Antigravity — mở bảng cài đặt MCP (MCP Store -> **View raw config**), tương ứng file `~/.gemini/config/mcp_config.json`, hoặc `.agents/mcp_config.json` nếu chỉ muốn áp dụng cho một workspace

```json
{
  "mcpServers": {
    "figma-mcp-android": {
      "command": "npx",
      "args": ["-y", "@impeterwayne/figma-mcp-android@latest"]
    }
  }
}
```

#### Codex CLI

`~/.codex/config.toml`:

```toml
[mcp_servers.figma-mcp-android]
command = "npx"
args = ["-y", "@impeterwayne/figma-mcp-android@latest"]
```

#### OpenCode
`opencode.json`
```json
{
    "$schema": "https://opencode.ai/config.json",
    "mcp": {
        "figma-mcp-android": {
            "type": "local",
            "command": [
                "npx",
                "-y",
                "@impeterwayne/figma-mcp-android@latest"
            ],
            "enabled": true
        }
    }
}
```

### 2. Cài plugin vào Figma Desktop

Không có plugin thì server không lấy được dữ liệu, nên bước này là bắt buộc.

1. Tải `figma-plugin.zip` mới nhất ở trang [releases](https://github.com/impeterwayne/figma_mcp_android/releases).
2. Giải nén ra một thư mục bất kỳ trên máy.
3. Mở Figma Desktop.
4. Chọn **Plugins** -> **Development** -> **Import plugin from manifest...**.
5. Trỏ tới file `manifest.json` trong thư mục vừa giải nén.
6. Mở file thiết kế cần làm việc rồi chạy plugin.
7. Mở AI client và bắt đầu gọi các tool của `figma-mcp-android`.

## Danh sách tool

21 tool, tất cả đều chỉ đọc file Figma, cộng thêm một tool chuyển đổi file trên máy.

### Duyệt tài liệu và node

| Tool | Chức năng |
|------|-----------|
| `get_design_context` | Trả về cây node rút gọn theo độ sâu, tiết kiệm token. **Nên dùng đầu tiên** khi file lớn. |
| `get_document` | Toàn bộ cây node của *page hiện tại* (không phải cả file). Đệ quy nên kết quả có thể rất lớn. |
| `get_pages` | Liệt kê các page kèm ID và tên — nhẹ hơn nhiều so với `get_document`. |
| `get_metadata` | Tên file, danh sách page và page đang mở. |
| `get_selection` | Các node đang chọn trong Figma; không chọn gì thì trả về mảng rỗng. |
| `get_node` | Lấy một node theo ID. ID viết theo dạng dấu hai chấm (`4029:12345`), không phải dấu gạch ngang. |
| `get_nodes_info` | Lấy nhiều node cùng lúc — nên dùng thay vì gọi `get_node` nhiều lần. |
| `search_nodes` | Tìm node theo một phần tên và/hoặc theo type, trong phạm vi một node cha. |
| `scan_nodes_by_types` | Lấy tất cả node thuộc những type chỉ định, bất kể tên là gì. |
| `scan_text_nodes` | Lấy toàn bộ nội dung chữ trong các node TEXT. |
| `get_reactions` | Các reaction prototype gắn trên node — trigger và danh sách action. |
| `get_viewport` | Vị trí khung nhìn hiện tại: tâm màn hình, mức zoom và vùng đang thấy. |
| `get_fonts` | Các font đang dùng trong page, sắp xếp theo mức độ sử dụng. |

### Style, variable, component, annotation, token

| Tool | Chức năng |
|------|-----------|
| `get_styles` | Các style paint, text, effect và grid trong file, kèm ID và thuộc tính. |
| `get_variable_defs` | Các variable trong file — collection, mode và giá trị (design token của Figma). |
| `get_local_components` | Các component được định nghĩa trong file hiện tại. |
| `get_annotations` | Annotation ở chế độ dev, lấy toàn page hoặc chỉ một node. |
| `export_tokens` | Xuất variable và paint style ra JSON hoặc CSS custom property. |

### Xuất file và công cụ cho Android

| Tool | Chức năng |
|------|-----------|
| `get_screenshot` | Chụp node và trả về ảnh dạng base64 trong bộ nhớ. |
| `save_screenshots` | Chụp node và lưu thẳng thành file, trả về đường dẫn, dung lượng, kích thước — không kèm base64. |
| `convert_svg_to_android_drawable` | Chuyển file SVG trên máy thành XML Android VectorDrawable, một hoặc nhiều file mỗi lần gọi. |

Cách xuất icon cho Android: gọi `save_screenshots` với `format: 'SVG'` trên node bọc icon (COMPONENT/FRAME, đừng chọn node VECTOR bên trong), rồi đưa đường dẫn file vừa lưu vào `svgPath` của `convert_svg_to_android_drawable`.

Vài điểm cần lưu ý với tool này:

- Chỉ nhận đường dẫn file, không nhận nội dung SVG dán trực tiếp hay base64.
- File SVG nguồn sẽ bị xóa sau khi chuyển xong, nên hãy xuất ra thư mục tạm, đừng trỏ vào SVG bạn còn cần giữ.
- Chuyển nhiều file thì chạy song song, một file lỗi không làm hỏng những file còn lại.
- Không dùng cho ảnh raster, layer effect hay blend mode — những thứ đó nên xuất PNG/WebP vào `res/drawable-*dpi`.

## Prompt

| Prompt | Mục đích |
|--------|----------|
| `read_design_strategy` | Cách đọc file Figma hiệu quả với server này. |
| `svg_to_drawable_strategy` | Quy trình đầy đủ từ vector trong Figma sang Android VectorDrawable. |
| `style_audit_strategy` | Rà soát thiết kế, tìm chỗ dùng giá trị thô thay vì style hoặc variable. |
| `reaction_to_connector_strategy` | Phân tích reaction prototype và dựng lại luồng tương tác. |

## Giấy phép

MIT. Xem [LICENSE](LICENSE).
