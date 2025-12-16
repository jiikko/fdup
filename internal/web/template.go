package web

import (
	"fmt"
	"strings"

	"github.com/jiikko/fdup/internal/code"
	"github.com/jiikko/fdup/internal/db"
)

func (s *Server) renderHTML(duplicateGroups []db.DuplicateGroup, currentPage, totalPages, totalGroups int) string {
	var groups strings.Builder
	for i, group := range duplicateGroups {
		groups.WriteString(fmt.Sprintf(`
		<div class="group" id="group-%d">
			<h2>%s <span class="count">%d files</span></h2>
			<ul>`, i, code.Format(group.Code), len(group.Files)))

		for j, file := range group.Files {
			groups.WriteString(fmt.Sprintf(`
				<li id="file-%d-%d" data-path="%s">
					<span class="path">%s</span>
					<span class="size">%s</span>
					<div class="actions">
						<button onclick="openFile('%s')" title="Open file">Open</button>
						<button onclick="revealFile('%s')" title="Reveal in Finder">Finder</button>
						<button onclick="deleteFile('%s', %d, %d)" class="delete" title="Move to Trash">Delete</button>
					</div>
				</li>`,
				i, j,
				escapeHTML(file.Path),
				escapeHTML(file.Path),
				formatSize(file.Size),
				escapeJS(file.Path),
				escapeJS(file.Path),
				escapeJS(file.Path), i, j))
		}

		groups.WriteString(`
			</ul>
		</div>`)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>fdup - Duplicate Files</title>
	<style>
		* {
			box-sizing: border-box;
		}
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
			max-width: 1200px;
			margin: 0 auto;
			padding: 20px;
			background: #f5f5f5;
		}
		header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			margin-bottom: 20px;
			padding-bottom: 10px;
			border-bottom: 1px solid #ddd;
		}
		h1 {
			margin: 0;
			color: #333;
		}
		.shutdown-btn {
			background: #dc3545;
			color: white;
			border: none;
			padding: 10px 20px;
			border-radius: 5px;
			cursor: pointer;
			font-size: 14px;
		}
		.shutdown-btn:hover {
			background: #c82333;
		}
		.group {
			background: white;
			border-radius: 8px;
			padding: 15px;
			margin-bottom: 15px;
			box-shadow: 0 1px 3px rgba(0,0,0,0.1);
		}
		.group h2 {
			margin: 0 0 10px 0;
			font-size: 18px;
			color: #333;
		}
		.group .count {
			font-size: 14px;
			color: #666;
			font-weight: normal;
		}
		ul {
			list-style: none;
			padding: 0;
			margin: 0;
		}
		li {
			display: flex;
			align-items: center;
			padding: 10px;
			border-bottom: 1px solid #eee;
			gap: 10px;
		}
		li:last-child {
			border-bottom: none;
		}
		li.deleted {
			opacity: 0.5;
			text-decoration: line-through;
			background: #fee;
		}
		.path {
			flex: 1;
			word-break: break-all;
			font-family: monospace;
			font-size: 13px;
		}
		.size {
			color: #666;
			font-size: 13px;
			white-space: nowrap;
		}
		.actions {
			display: flex;
			gap: 5px;
		}
		button {
			padding: 5px 10px;
			border: 1px solid #ddd;
			background: #fff;
			border-radius: 4px;
			cursor: pointer;
			font-size: 12px;
		}
		button:hover {
			background: #f0f0f0;
		}
		button.delete {
			color: #dc3545;
			border-color: #dc3545;
		}
		button.delete:hover {
			background: #dc3545;
			color: white;
		}
		button:disabled {
			opacity: 0.5;
			cursor: not-allowed;
		}
		.toast {
			position: fixed;
			bottom: 20px;
			right: 20px;
			padding: 15px 20px;
			border-radius: 5px;
			color: white;
			font-size: 14px;
			opacity: 0;
			transition: opacity 0.3s;
			z-index: 1000;
		}
		.toast.show {
			opacity: 1;
		}
		.toast.success {
			background: #28a745;
		}
		.toast.error {
			background: #dc3545;
		}
		.summary {
			color: #666;
			margin-bottom: 20px;
		}
		.pagination {
			display: flex;
			justify-content: center;
			gap: 5px;
			margin: 20px 0;
		}
		.pagination a, .pagination span {
			padding: 8px 12px;
			border: 1px solid #ddd;
			border-radius: 4px;
			text-decoration: none;
			color: #333;
		}
		.pagination a:hover {
			background: #f0f0f0;
		}
		.pagination .current {
			background: #007bff;
			color: white;
			border-color: #007bff;
		}
		.pagination .disabled {
			color: #999;
			cursor: not-allowed;
		}
	</style>
</head>
<body>
	<header>
		<h1>fdup - Duplicate Files</h1>
		<button class="shutdown-btn" onclick="shutdown()">Shutdown Server</button>
	</header>
	<p class="summary">Found %d duplicate groups (showing page %d of %d)</p>
	%s
	%s
	<div id="toast" class="toast"></div>
	<script>
		function showToast(message, type) {
			const toast = document.getElementById('toast');
			toast.textContent = message;
			toast.className = 'toast ' + type + ' show';
			setTimeout(() => {
				toast.className = 'toast';
			}, 3000);
		}

		async function apiCall(endpoint, path) {
			try {
				const response = await fetch('/api/' + endpoint, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ path: path })
				});
				return await response.json();
			} catch (e) {
				return { status: 'error', message: e.message };
			}
		}

		async function openFile(path) {
			const result = await apiCall('open', path);
			if (result.status === 'ok') {
				showToast('File opened', 'success');
			} else {
				showToast('Error: ' + result.message, 'error');
			}
		}

		async function revealFile(path) {
			const result = await apiCall('reveal', path);
			if (result.status === 'ok') {
				showToast('Revealed in Finder', 'success');
			} else {
				showToast('Error: ' + result.message, 'error');
			}
		}

		async function deleteFile(path, groupIdx, fileIdx) {
			if (!confirm('Move this file to Trash?\\n\\n' + path)) {
				return;
			}

			const result = await apiCall('delete', path);
			if (result.status === 'ok') {
				const li = document.getElementById('file-' + groupIdx + '-' + fileIdx);
				if (li) {
					li.classList.add('deleted');
					li.querySelectorAll('button').forEach(btn => btn.disabled = true);
				}
				showToast('Moved to Trash', 'success');
			} else {
				showToast('Error: ' + result.message, 'error');
			}
		}

		async function shutdown() {
			if (!confirm('Shutdown the server?')) {
				return;
			}
			await fetch('/api/shutdown', { method: 'POST' });
			showToast('Server shutting down...', 'success');
			setTimeout(() => {
				document.body.innerHTML = '<h1 style="text-align:center;margin-top:100px;">Server stopped. You can close this tab.</h1>';
			}, 500);
		}
	</script>
</body>
</html>`, totalGroups, currentPage, totalPages, groups.String(), renderPagination(currentPage, totalPages))
}

func renderPagination(currentPage, totalPages int) string {
	if totalPages <= 1 {
		return ""
	}

	var b strings.Builder
	b.WriteString(`<div class="pagination">`)

	// Previous
	if currentPage > 1 {
		b.WriteString(fmt.Sprintf(`<a href="?page=%d">&laquo; Prev</a>`, currentPage-1))
	} else {
		b.WriteString(`<span class="disabled">&laquo; Prev</span>`)
	}

	// Page numbers
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			b.WriteString(fmt.Sprintf(`<span class="current">%d</span>`, i))
		} else if i == 1 || i == totalPages || (i >= currentPage-2 && i <= currentPage+2) {
			b.WriteString(fmt.Sprintf(`<a href="?page=%d">%d</a>`, i, i))
		} else if i == currentPage-3 || i == currentPage+3 {
			b.WriteString(`<span>...</span>`)
		}
	}

	// Next
	if currentPage < totalPages {
		b.WriteString(fmt.Sprintf(`<a href="?page=%d">Next &raquo;</a>`, currentPage+1))
	} else {
		b.WriteString(`<span class="disabled">Next &raquo;</span>`)
	}

	b.WriteString(`</div>`)
	return b.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func escapeJS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
