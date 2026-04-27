'use client';

import { useTranslations } from 'next-intl';

import { AuthGate } from '@/components/auth-gate';
import { PageHero } from '@/components/page-hero';

import { useOnboarding } from './_hooks/use-onboarding';
import { OnboardingMetrics } from './_components/onboarding-metrics';
import { OnboardingPathForm } from './_components/onboarding-path-form';
import { OnboardingTaskForm } from './_components/onboarding-task-form';
import { OnboardingPathTable } from './_components/onboarding-path-table';
import { OnboardingTaskTable } from './_components/onboarding-task-table';

export default function OnboardingPage() {
  const pageT = useTranslations('pages');
  const h = useOnboarding();

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Onboarding" title={pageT('onboardingTitle')} description={pageT('onboardingDesc')} />
        <OnboardingMetrics {...h.onboardingSummary} />
        <section className="forms-grid">
          <OnboardingPathForm
            t={h.t} pathForm={h.pathForm} pathMode={h.pathMode}
            pathModeOptions={h.pathModeOptions} nodes={h.nodes}
            createPathMutation={h.createPathMutation}
          />
          <OnboardingTaskForm
            t={h.t} taskForm={h.taskForm} taskMode={h.taskMode}
            pathModeOptions={h.pathModeOptions} paths={h.paths} nodes={h.nodes}
            createTaskMutation={h.createTaskMutation}
          />
        </section>
        <OnboardingPathTable
          t={h.t} pathsQuery={h.pathsQuery} paths={h.filteredPaths}
          totalPaths={h.paths.length}
          nodesByID={h.nodesByID} taskCountByPathID={h.taskCountByPathID}
          taskSummaryByPathID={h.taskSummaryByPathID} editingPath={h.editingPath}
          pathState={{
            value: h.pathQuery, set: h.setPathQuery,
            modeFilter: h.pathModeFilter, setModeFilter: h.setPathModeFilter,
            enabledFilter: h.pathEnabledFilter, setEnabledFilter: h.setPathEnabledFilter,
            editingPathID: h.editingPathID, setEditingPathID: h.setEditingPathID,
            pathEditorState: h.pathEditorState, setPathEditorState: h.setPathEditorState,
          }}
          deletePathMutation={h.deletePathMutation}
          pathModeOptions={h.pathModeOptions}
          updatePathMutation={h.updatePathMutation}
          nodes={h.nodes}
        />
        <OnboardingTaskTable
          t={h.t} tasksQuery={h.tasksQuery} tasks={h.filteredTasks}
          totalTasks={h.tasks.length}
          nodesByID={h.nodesByID} pathsByID={h.pathsByID} enums={h.enums}
          availableTaskStatuses={h.availableTaskStatuses}
          pathModeOptions={h.pathModeOptions} editingTask={h.editingTask}
          taskState={{
            value: h.taskQuery, set: h.setTaskQuery,
            modeFilter: h.taskModeFilter, setModeFilter: h.setTaskModeFilter,
            statusFilter: h.taskStatusFilter, setStatusFilter: h.setTaskStatusFilter,
            editingTaskID: h.editingTaskID, setEditingTaskID: h.setEditingTaskID,
            taskEditorState: h.taskEditorState, setTaskEditorState: h.setTaskEditorState,
          }}
          taskStatusOptions={h.taskStatusOptions}
          updateTaskStatusMutation={h.updateTaskStatusMutation}
        />
      </div>
    </AuthGate>
  );
}
