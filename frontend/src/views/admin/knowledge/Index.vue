<template>
  <div class="knowledge-container admin-page-shell">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">知识库</h2>
        <p class="page-subtitle">维护帮助中心的分类与文章，用户端「帮助中心」实时读取</p>
      </div>
    </div>

    <n-card :bordered="false">
      <n-tabs type="line" v-model:value="tab">
        <n-tab-pane name="articles" tab="知识库文章">
          <n-space vertical>
            <n-space>
              <n-button type="primary" @click="handleAddArticle">新增文章</n-button>
              <n-select v-model:value="filterCategory" :options="categoryFilterOptions" clearable placeholder="按分类筛选" style="width: 200px" />
            </n-space>
            <n-data-table :columns="articleColumns" :data="articles" :loading="loadingArticles" :pagination="{ pageSize: 20 }" size="small" />
          </n-space>
        </n-tab-pane>
        <n-tab-pane name="categories" tab="分类管理">
          <n-space vertical>
            <n-button type="primary" @click="handleAddCategory">新增分类</n-button>
            <n-data-table :columns="categoryColumns" :data="categories" :loading="loadingCategories" size="small" />
          </n-space>
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <!-- 文章编辑 -->
    <n-modal v-model:show="showArticleModal" preset="card" :title="editingArticle ? '编辑文章' : '新增文章'" style="width: 720px">
      <n-form :model="articleForm" label-placement="left" label-width="80">
        <n-form-item label="分类">
          <n-select v-model:value="articleForm.category_id" :options="categoryOptions" />
        </n-form-item>
        <n-form-item label="标题"><n-input v-model:value="articleForm.title" /></n-form-item>
        <n-form-item label="摘要"><n-input v-model:value="articleForm.summary" /></n-form-item>
        <n-form-item label="内容">
          <n-input v-model:value="articleForm.content" type="textarea" :rows="10" />
        </n-form-item>
        <n-form-item label="排序"><n-input-number v-model:value="articleForm.sort_order" :min="0" /></n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showArticleModal = false">取消</n-button>
          <n-button type="primary" :loading="savingArticle" @click="saveArticle">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 分类编辑 -->
    <n-modal v-model:show="showCategoryModal" preset="card" :title="editingCategory ? '编辑分类' : '新增分类'" style="width: 420px">
      <n-form :model="categoryForm" label-placement="left" label-width="70">
        <n-form-item label="名称"><n-input v-model:value="categoryForm.name" /></n-form-item>
        <n-form-item label="图标"><n-input v-model:value="categoryForm.icon" placeholder="可选" /></n-form-item>
        <n-form-item label="排序"><n-input-number v-model:value="categoryForm.sort_order" :min="0" /></n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCategoryModal = false">取消</n-button>
          <n-button type="primary" :loading="savingCategory" @click="saveCategory">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NButton, NSpace, useMessage } from 'naive-ui'
import request from '@/utils/request'

const message = useMessage()
const tab = ref('articles')
const articles = ref<any[]>([])
const categories = ref<any[]>([])
const loadingArticles = ref(false)
const loadingCategories = ref(false)
const filterCategory = ref<number | null>(null)

const showArticleModal = ref(false)
const editingArticle = ref(false)
const editingArticleId = ref(0)
const savingArticle = ref(false)
const articleForm = reactive<any>({ category_id: null, title: '', summary: '', content: '', sort_order: 0 })

const showCategoryModal = ref(false)
const editingCategory = ref(false)
const editingCategoryId = ref(0)
const savingCategory = ref(false)
const categoryForm = reactive<any>({ name: '', icon: '', sort_order: 0 })

const categoryOptions = computed(() => categories.value.map(c => ({ label: c.name, value: c.id })))
const categoryFilterOptions = categoryOptions

async function loadCategories() {
  loadingCategories.value = true
  try {
    const res: any = await request.get('/admin/knowledge/categories')
    categories.value = res?.data || []
  } finally {
    loadingCategories.value = false
  }
}

async function loadArticles() {
  loadingArticles.value = true
  try {
    const params: any = {}
    if (filterCategory.value) params.category_id = filterCategory.value
    const res: any = await request.get('/admin/knowledge/articles', { params })
    articles.value = res?.data || []
  } finally {
    loadingArticles.value = false
  }
}

function handleAddArticle() {
  editingArticle.value = false
  editingArticleId.value = 0
  Object.assign(articleForm, { category_id: categories.value[0]?.id || null, title: '', summary: '', content: '', sort_order: 0 })
  showArticleModal.value = true
}

function handleEditArticle(row: any) {
  editingArticle.value = true
  editingArticleId.value = row.id
  Object.assign(articleForm, {
    category_id: row.category_id, title: row.title, summary: row.summary || '',
    content: row.content, sort_order: row.sort_order,
  })
  showArticleModal.value = true
}

async function saveArticle() {
  if (!articleForm.category_id) { message.warning('请选择分类'); return }
  if (!articleForm.title) { message.warning('请填写标题'); return }
  savingArticle.value = true
  try {
    if (editingArticle.value) {
      await request.put(`/admin/knowledge/articles/${editingArticleId.value}`, articleForm)
    } else {
      await request.post('/admin/knowledge/articles', articleForm)
    }
    message.success('保存成功')
    showArticleModal.value = false
    await loadArticles()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '保存失败')
  } finally {
    savingArticle.value = false
  }
}

async function handleDeleteArticle(row: any) {
  try {
    await request.delete(`/admin/knowledge/articles/${row.id}`)
    message.success('已删除')
    await loadArticles()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '删除失败')
  }
}

function handleAddCategory() {
  editingCategory.value = false
  editingCategoryId.value = 0
  Object.assign(categoryForm, { name: '', icon: '', sort_order: 0 })
  showCategoryModal.value = true
}

function handleEditCategory(row: any) {
  editingCategory.value = true
  editingCategoryId.value = row.id
  Object.assign(categoryForm, { name: row.name, icon: row.icon || '', sort_order: row.sort_order })
  showCategoryModal.value = true
}

async function saveCategory() {
  if (!categoryForm.name) { message.warning('请填写分类名称'); return }
  savingCategory.value = true
  try {
    if (editingCategory.value) {
      await request.put(`/admin/knowledge/categories/${editingCategoryId.value}`, categoryForm)
    } else {
      await request.post('/admin/knowledge/categories', categoryForm)
    }
    message.success('保存成功')
    showCategoryModal.value = false
    await loadCategories()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '保存失败')
  } finally {
    savingCategory.value = false
  }
}

async function handleDeleteCategory(row: any) {
  try {
    await request.delete(`/admin/knowledge/categories/${row.id}`)
    message.success('已删除')
    await loadCategories()
    await loadArticles()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '删除失败')
  }
}

const articleColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '标题', key: 'title', ellipsis: { tooltip: true } },
  { title: '分类', key: 'category_id', width: 120,
    render: (row: any) => categories.value.find(c => c.id === row.category_id)?.name || '-' },
  { title: '浏览量', key: 'view_count', width: 80 },
  { title: '操作', key: 'actions', width: 140,
    render: (row: any) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'tiny', onClick: () => handleEditArticle(row) }, { default: () => '编辑' }),
        h(NButton, { size: 'tiny', type: 'error', secondary: true, onClick: () => handleDeleteArticle(row) }, { default: () => '删除' }),
      ],
    }) },
]

const categoryColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name' },
  { title: '图标', key: 'icon', width: 100 },
  { title: '排序', key: 'sort_order', width: 80 },
  { title: '操作', key: 'actions', width: 140,
    render: (row: any) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'tiny', onClick: () => handleEditCategory(row) }, { default: () => '编辑' }),
        h(NButton, { size: 'tiny', type: 'error', secondary: true, onClick: () => handleDeleteCategory(row) }, { default: () => '删除' }),
      ],
    }) },
]

onMounted(async () => {
  await loadCategories()
  await loadArticles()
})
</script>
