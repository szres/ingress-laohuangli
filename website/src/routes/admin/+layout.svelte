<script>
	import { page } from '$app/stores';

	$: currentPath = $page.url.pathname;
</script>

<div class="flex h-full">
	<!-- 侧边栏 -->
	<div class="w-56 bg-base-200 p-4 flex flex-col gap-2">
		<h2 class="text-lg font-bold mb-4 px-2">管理后台</h2>
		<ul class="menu menu-lg gap-1">
			<li>
				<a href="/admin" class:active={currentPath === '/admin'}> ⚙️ 配置管理 </a>
			</li>
			<li>
				<a href="/admin/logs" class:active={currentPath === '/admin/logs'}> 📋 日志查看 </a>
			</li>
			<li>
				<a href="/admin/tokens" class:active={currentPath === '/admin/tokens'}> 🔑 API Token </a>
			</li>
			<li>
				<a href="/admin/ai-results" class:active={currentPath === '/admin/ai-results'}> ✨ AI 结果 </a>
			</li>
			<li>
				<a href="/admin/ai-library" class:active={currentPath === '/admin/ai-library'}> 📚 AI 词条库 </a>
			</li>
		</ul>
		<div class="divider"></div>
		<a href="/" class="btn btn-ghost btn-sm">← 返回首页</a>
	</div>

	<!-- 主内容区 -->
	<div class="flex-1 overflow-auto p-6">
		<slot />
	</div>
</div>
