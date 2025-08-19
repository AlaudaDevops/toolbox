import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { roadmapAPI, handleAPIError } from '../services/api';
import { sortRoadmapData } from '../utils/sortingUtils';
import toast from 'react-hot-toast';

const RoadmapContext = createContext();

export const useRoadmap = () => {
  const context = useContext(RoadmapContext);
  if (!context) {
    throw new Error('useRoadmap must be used within a RoadmapProvider');
  }
  return context;
};

export const RoadmapProvider = ({ children }) => {
  const [roadmapData, setRoadmapData] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  // Load basic data first (fast) - simplified approach
  const loadBasicData = useCallback(async () => {
    try {
      const basicData = await roadmapAPI.getBasicData();
      console.log('Basic data loaded:', basicData); // Debug log

      // Set basic data immediately for fast UI rendering
      setRoadmapData(prevData => ({
        ...prevData,
        pillars: basicData.pillars.map(pillar => ({
          ...pillar,
          milestones: [] // Will be loaded with full data
        })),
        quarters: basicData.quarters,
        components: basicData.components,
        versions: basicData.versions,
      }));

      return basicData;
    } catch (error) {
      console.warn('Failed to load basic data, will load full data:', error);
      return null;
    }
  }, []);

  // Load roadmap data (simplified approach)
  const loadRoadmap = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);

      // Try to load basic data first for fast UI feedback
      await loadBasicData();

      // Then load complete data using the original API
      const data = await roadmapAPI.getRoadmap();
      console.log('Complete roadmap data loaded:', data); // Debug log

      // Sort the roadmap data
      const sortedData = sortRoadmapData(data);
      console.log('Roadmap data sorted:', sortedData); // Debug log

      setRoadmapData(sortedData);
    } catch (error) {
      const errorInfo = handleAPIError(error);
      console.error('Failed to load roadmap:', errorInfo); // Debug log
      setError(errorInfo.message);
      toast.error(`Failed to load roadmap: ${errorInfo.message}`);
    } finally {
      setIsLoading(false);
    }
  }, [loadBasicData]);

  // Load roadmap on mount
  useEffect(() => {
    loadRoadmap();
  }, [loadRoadmap]);

  // Create milestone
  const createMilestone = async (milestoneData) => {
    try {
      const milestone = await roadmapAPI.createMilestone(milestoneData);

      // Optimistically update the roadmap data
      setRoadmapData(prevData => {
        if (!prevData) return prevData;

        const updatedPillars = prevData.pillars.map(pillar => {
          if (pillar.id === milestoneData.pillar_id) {
            return {
              ...pillar,
              milestones: [...pillar.milestones, milestone],
            };
          }
          return pillar;
        });

        return {
          ...prevData,
          pillars: updatedPillars,
        };
      });

      toast.success('Milestone created successfully!');
      return { success: true, data: milestone };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to create milestone: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Update milestone
  const updateMilestone = async (milestoneId, milestoneData) => {
    try {
      const response = await roadmapAPI.updateMilestone(milestoneId, milestoneData);

      // Reload roadmap data to get the updated state
      await loadRoadmap();

      toast.success('Milestone updated successfully!');
      return { success: true, data: response };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to update milestone: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Create epic
  const createEpic = async (epicData) => {
    try {
      const epic = await roadmapAPI.createEpic(epicData);

      // Optimistically update the roadmap data
      setRoadmapData(prevData => {
        if (!prevData) return prevData;

        const updatedPillars = prevData.pillars.map(pillar => ({
          ...pillar,
          milestones: pillar.milestones.map(milestone => {
            if (milestone.id === epicData.milestone_id) {
              return {
                ...milestone,
                epics: [...milestone.epics, epic],
              };
            }
            return milestone;
          }),
        }));

        return {
          ...prevData,
          pillars: updatedPillars,
        };
      });

      toast.success('Epic created successfully!');
      return { success: true, data: epic };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to create epic: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Update epic
  const updateEpic = async (epicId, epicData) => {
    try {
      const response = await roadmapAPI.updateEpic(epicId, epicData);

      // Reload roadmap data to get the updated state
      await loadRoadmap();

      toast.success('Epic updated successfully!');
      return { success: true, data: response };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to update epic: ${errorInfo.message}`);
      return { success: false, error: errorInfo.message };
    }
  };

  // Move epic to different milestone
  const moveEpic = async (epicId, newMilestoneId) => {
    try {
      await roadmapAPI.updateEpicMilestone(epicId, newMilestoneId);

      // Optimistically update the roadmap data
      setRoadmapData(prevData => {
        if (!prevData) return prevData;

        let movedEpic = null;

        // Remove epic from current milestone
        const updatedPillars = prevData.pillars.map(pillar => ({
          ...pillar,
          milestones: pillar.milestones.map(milestone => ({
            ...milestone,
            epics: milestone.epics.filter(epic => {
              if (epic.id === epicId) {
                movedEpic = { ...epic, milestone_id: newMilestoneId };
                return false;
              }
              return true;
            }),
          })),
        }));

        // Add epic to new milestone
        const finalPillars = updatedPillars.map(pillar => ({
          ...pillar,
          milestones: pillar.milestones.map(milestone => {
            if (milestone.id === newMilestoneId && movedEpic) {
              return {
                ...milestone,
                epics: [...milestone.epics, movedEpic],
              };
            }
            return milestone;
          }),
        }));

        return {
          ...prevData,
          pillars: finalPillars,
        };
      });

      toast.success('Epic moved successfully!');
      return { success: true };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      toast.error(`Failed to move epic: ${errorInfo.message}`);
      // Reload roadmap to ensure consistency
      loadRoadmap();
      return { success: false, error: errorInfo.message };
    }
  };

  // Get component versions
  const getComponentVersions = async (componentName) => {
    try {
      const response = await roadmapAPI.getComponentVersions(componentName);
      return { success: true, data: response.versions };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      return { success: false, error: errorInfo.message };
    }
  };

  // Get assignable users
  const getAssignableUsers = async (issueKey, query = '') => {
    try {
      const response = await roadmapAPI.getAssignableUsers(issueKey, query);
      return { success: true, data: response.users };
    } catch (error) {
      const errorInfo = handleAPIError(error);
      return { success: false, error: errorInfo.message };
    }
  };

  const value = {
    roadmapData,
    isLoading,
    error,
    loadRoadmap,
    loadBasicData,
    createMilestone,
    updateMilestone,
    createEpic,
    updateEpic,
    moveEpic,
    getComponentVersions,
    getAssignableUsers,
  };

  return (
    <RoadmapContext.Provider value={value}>
      {children}
    </RoadmapContext.Provider>
  );
};
