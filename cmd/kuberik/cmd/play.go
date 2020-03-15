/*
Copyright © 2020 none

*/
package cmd

import (
	"fmt"

	"strings"

	corev1alpha1 "github.com/kuberik/kuberik/pkg/apis/core/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	playFrom *string
	playVars *[]string
)

func init() {
	createCmd.AddCommand(playCmd)
	playFrom = playCmd.Flags().String("from", "", "Create a Play from Movie spec")
	playCmd.MarkFlagRequired("from")
	playVars = playCmd.Flags().StringSlice("var", []string{}, "Additional Play vars")
}

// playCmd represents the play command
var playCmd = &cobra.Command{
	Use:   "play",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(0),
	Run:  createPlay,
}

func createPlay(cmd *cobra.Command, args []string) {
	vars, err := parseVars(*playVars...)
	movie, err := client.Movies(namespace).Get(*playFrom, v1.GetOptions{})
	if err != nil {
		cmd.PrintErr(err)
		return
	}

	play, err := PlayFromMovie(movie, vars)
	if err != nil {
		cmd.PrintErr(err)
		return
	}

	if instance, err := CreatePlayInstance(play); err != nil {
		cmd.PrintErrf("Failed to create play: %s", err)
	} else {
		cmd.Printf("play/%s created\n", instance.Name)
	}
}

func PlayFromMovie(movie *corev1alpha1.Movie, vars []corev1alpha1.Var) (*corev1alpha1.Play, error) {
	play := playFromMovie(movie)
	err := populateVars(&play, vars...)

	if err != nil {
		return nil, err
	}
	return &play, err
}

func parseVars(varInputs ...string) (vars []corev1alpha1.Var, err error) {
	for _, v := range varInputs {
		splitVar := strings.SplitN(v, "=", 2)

		if len(splitVar) != 2 {
			return nil, fmt.Errorf("Wrong variable format: %s", v)
		}

		vars = append(vars, corev1alpha1.Var{
			Name:  splitVar[0],
			Value: splitVar[1],
		})
	}
	return vars, nil
}

func populateVars(play *corev1alpha1.Play, vars ...corev1alpha1.Var) error {
	for _, v := range vars {

		if err := play.Spec.Screenplay.Vars.Set(v.Name, v.Value); err != nil {
			return err
		}
	}
	return nil
}

func playFromMovie(movie *corev1alpha1.Movie) corev1alpha1.Play {
	return corev1alpha1.Play{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", movie.Name),
			Namespace:    movie.Namespace,
		},
		Spec: corev1alpha1.PlaySpec{
			Screenplay: *movie.Spec.Screenplay,
		},
		Status: corev1alpha1.PlayStatus{
			Phase:  corev1alpha1.PlayCreated,
			Frames: make(map[string]int),
		},
	}
}

func CreatePlayInstance(play *corev1alpha1.Play) (instance *corev1alpha1.Play, err error) {
	instance, err = client.Plays(play.Namespace).Create(play)
	return
}
